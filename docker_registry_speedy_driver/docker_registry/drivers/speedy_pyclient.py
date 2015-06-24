
import gevent
import requests
import random
import operator
import threading
import logging
from docker_registry.core import exceptions

global AllUrls
global HealthUrls
global HeartBeatMutex

logger = logging.getLogger(__name__)

def init_speedy(imageservers):
    global AllUrls
    global HealthUrls
    global HeartBeatMutex

    AllUrls = imageservers.split(';')
    HealthUrls = imageservers.split(';')
    HeartBeatMutex = threading.Lock()

class HTTPConnection:
    def __init__(self, url):
        self.url = url

    def _url(self, full_path):
        return "%s/%s" % (self.url, full_path)

    def post(self, full_path, headers=None, data=None):
        if headers is None:
            headers = {}

        url = self._url(full_path)
        return requests.post(url, data=data, headers=headers)

    def get(self, full_path, headers=None, stream=False):
        if headers is None:
            headers = {}

        url = self._url(full_path)
        return requests.get(url, headers=headers, stream=stream)

    def delete(self, full_path, headers=None):
        if headers is None:
            headers = {}

        url = self._url(full_path)
        return requests.delete(url, headers=headers)

class Connection(object):
    max_retry_times = 5

    def __init__(self):
        pass

    def _retry(self, func, *args, **kwargs):
        retry_times = 0
        while True:
            try:
                rv = func(*args, **kwargs)
                return rv
            except:
                retry_times += 1
                if retry_times > self.max_retry_times:
                    raise exceptions.UnspecifiedError("retry %s failed" % func.__name__)

    def _gen_http_conn(self):
        """
        @choose url and generate connection
        """

        global HealthUrls
        global HeartBeatMutex

        url = ""

        HeartBeatMutex.acquire()
        size = len(HealthUrls)
        if size > 0:
            index = random.randint(0, size-1)
            url = HealthUrls[index]
        HeartBeatMutex.release()

        if len(url) == 0:
            logger.error("no healthy image server")
            raise exceptions.UnspecifiedError("No Healthy ImageServer!!!")

        return HTTPConnection(url)

    def _check_fragments(self, fragments):
        if not isinstance(fragments, list):
            return None, None

        fragcount = 0
        max_range = 0
        sorted_fragments = sorted(fragments, key=operator.itemgetter('Index'))
        for fragment in sorted_fragments:
            if fragment['Index'] != fragcount or fragment['Start'] != max_range:
                return None, None
            max_range = fragment['End']
            fragcount += 1

        last_fragment = sorted_fragments[-1]
        if not last_fragment['IsLast']:
            raise exceptions.UnspecifiedError("uncompleted file fragment infos")

        return max_range, sorted_fragments

    def _exists(self, path):
        conn = self._gen_http_conn()

        headers = {}
        headers["Path"] = path

        return conn.get("v1/fileinfo", headers=headers)

    def exists(self, path):
        logger.debug("exists, path: %s" % path)

        resp = self._retry(self._exists, path)

        if resp.status_code == 200:
            return True
        elif resp.status_code == 404:
            return False
        else:
            logger.error("Unkown stauts code: %d" % resp.status_code)
            raise exceptions.UnspecifiedError("unexcept status code: %d" % resp.status_code)

    def _getfileinfo(self, path):
        conn = self._gen_http_conn()

        headers = {}
        headers["Path"] = path

        return conn.get("v1/fileinfo", headers=headers)

    def getfileinfo(self, path):
        logger.debug("getfileinfo: %s" % path)

        resp = self._retry(self._getfileinfo, path)

        if resp.status_code == 200:
            j = resp.json()

            key = "fragment-info"
            if key in j:
                fragementsinfo = j[key]
                max_range, fragementsinfo = self._check_fragments(fragementsinfo)
                if not max_range or not fragementsinfo:
                    raise exceptions.UnspecifiedError("fragment-info Error!")
                return max_range, fragementsinfo
            else:
                raise exceptions.UnspecifiedError("fileinfo not contain fragment-info!")

        elif resp.status_code == 404:
            raise exceptions.FileNotFoundError("File Not Found!")
        else:
            raise exceptions.UnspecifiedError("getfileinfo UnKnow status code:%d"
                                              % resp.status_code)

    def _upload(self, path, data=None, fragment_index=None, bytes_range=None, is_last=False):
        conn = self._gen_http_conn()

        headers = {}
        headers["Path"] = path
        headers["Fragment-Index"] = str(fragment_index)
        headers["Bytes-Range"] = "%s-%s" % (bytes_range[0], bytes_range[1])
        headers["Is-Last"] = "true" if is_last else "false"
        headers["Registry-Version"] = "v1"

        return conn.post("v1/file", data=data, headers=headers)

    def upload(self, path, data=None, fragment_index=None, bytes_range=None, is_last=False):
        logger.debug("upload path: %s, fragment_Index: %d" % (path, fragment_index))

        return self._retry(self._upload, path, data=data, fragment_index=fragment_index,
                           bytes_range=bytes_range, is_last=is_last)

    def _download(self, path, fragment_index, bytes_range, stream=False):
        conn = self._gen_http_conn()

        headers = {}
        headers["Path"] = path
        headers["Fragment-Index"] = str(fragment_index)
        headers["Bytes-Range"] = "%s-%s" % (bytes_range[0], bytes_range[1])

        return conn.get("v1/file", headers=headers, stream=stream)

    def download(self, path, fragment_index, bytes_range, stream=False):
        logger.debug("download path: %s, fragment_index: %d" % (path, fragment_index))

        return self._retry(self._download, path, fragment_index, bytes_range, stream=stream)

    def _delete(self, path):
        conn = self._gen_http_conn()

        headers = {}
        headers["Path"] = path
        headers["Registry-Version"] = "v1"

        return conn.delete("v1/file", headers=headers)

    def delete(self, path):
        logger.debug("delete path: %s" % path)

        return self._retry(self._delete, path)

    def _list_directory(self, path):
        conn = self._gen_http_conn()

        headers = {}
        headers["Path"] = path

        return conn.get("v1/list_directory", headers=headers)

    def list_directory(self, path):
        logger.debug("list directory path: %s" % path)

        return self._retry(self._list_directory, path)

def speedy_heart_beater(interval):

    global AllUrls
    global HealthUrls
    global HeartBeatMutex

    while True:
        health_urls = []

        # heart beat
        for url in AllUrls:
            try:
                conn = HTTPConnection(url)
                headers = {}
                headers["Path"] = "heartbeat"
                resp = conn.post("v1/_ping", headers=headers)

                if resp.status_code == 200 or resp.status_code == 404:
                    health_urls.append(url)
                else:
                    logger.error("unexcept resp code: %d" % resp.status_code)
            except:
                logger.error("[HeartBeater] connot connect to host : %s" % url)

        # update health urls
        HeartBeatMutex.acquire()
        HealthUrls = health_urls
        HeartBeatMutex.release()

        logger.debug("health urls: %s" % HealthUrls)

        gevent.sleep(interval)
