# -*- coding: utf-8 -*-
"""
docker_registry.drivers.speedy
~~~~~~~~~~~~~~~~~~~~~~~~~~

This is a speedy based driver.

"""

import gevent
import threading
import os
import random
import string
import logging
import time

from . import speedy_pyclient
from docker_registry.core import driver
from docker_registry.core import exceptions

logger = logging.getLogger(__name__)

class _TempFile(object):
    def __init__(self, mode='w+b', prefix='/tmp/'):
        self.prefix = prefix
        self.name = self._gen_file_name()
        self.file = None
        if self.name:
            self.file = open(self.name, mode)

    def _gen_file_name(self):
        while True:
            name = self.prefix + string.join(random.sample(['z','y','x','w','v','u','t','s','r','q','p','o',
                                   'n','m','l','k','j','i','h','g','f','e','d','c',
                                   'b','a'], 5)).replace(' ','')
            if os.path.exists(name):
                continue
            else:
                return name

    def close(self):
        if self.file:
            self.file.close()
            os.remove(self.name)

class _SpeedyMultiPartUploadContext(object):
    def __init__(self, path, conn, fragment_size, tmpdir):
        self.path = path
        self.conn = conn
        self.fragment_size = fragment_size
        self.tmpdir = tmpdir

        self.fragment_tmp_file = []
        self.cur_fragment = 0
        self.cur_offset = 0

        self._lock = threading.Lock()
        self.success_parts = 0
        self.failed_parts = 0

    def _refresh_part_completed(self, status):
        try:
            self._lock.acquire()

            if status == 0:
                self.success_parts += 1
            else:
                self.failed_parts += 1
        finally:
            self._lock.release()

    def _upload_part(self, fragment_index, bytes_range, last_fragment=False):
        logger.debug("spawn _upload_part, path:%s, index:%d, bytesrange:%d-%d" % \
                     (self.path, fragment_index, bytes_range[0], bytes_range[1]))

        f = self.fragment_tmp_file[fragment_index].file
        f.seek(0)

        try:
            resp = self.conn.upload(self.path, data=f, fragment_index=fragment_index, bytes_range=bytes_range,
                                    is_last=last_fragment)

            if resp.status_code == 200:
                logger.debug("_upload_part success, path:%s, index: %d" % \
                             (self.path, fragment_index))
                self._refresh_part_completed(0)
            else:
                logger.warning("_upload_part failed, path: %s, index: %d, statuscode: %d" % \
                            (self.path, fragment_index, resp.status_code))
                self._refresh_part_completed(1)

        except:
            logger.error("upload part failed, path:%s, index:%d, bytesrange:%d-%d" % \
                         (self.path, fragment_index, bytes_range[0], bytes_range[1]))
            self._refresh_part_completed(2)

        # delete tmp file after uploaded
        self.fragment_tmp_file[fragment_index].close()
        self.fragment_tmp_file[fragment_index] = None

        logger.debug("exit upload_part, path: %s, fragment: %d" % (self.path, fragment_index))

    def push_content(self, buf, more_data=True):
        if len(self.fragment_tmp_file) <= self.cur_fragment:
            self.fragment_tmp_file.append(_TempFile(mode='w+b', prefix=self.tmpdir))

        f = self.fragment_tmp_file[self.cur_fragment].file
        # seek to file end, and write data to file
        f.seek(0, 2)
        f.write(buf)
        fsize = f.tell()
        #print "after push_content, fsize:%d" % fsize
        logger.debug("after push content fsize: %d" % fsize)
        if fsize >= self.fragment_size or not more_data:
            # upload this fragment
            f.flush()

            logger.debug("begin spawn upload part, fragment: %d" % self.cur_fragment)
            gevent.spawn(self._upload_part, self.cur_fragment, (self.cur_offset, self.cur_offset + fsize),
                        not more_data)
            logger.debug("end spawn upload part, fragment: %d" % self.cur_fragment)

            self.cur_offset += fsize
            self.cur_fragment += 1

    def _check_error(self):
        if self.failed_parts > 0:
            self.conn.delete(self.path)
            raise exceptions.UnspecifiedError("speedy upload failed")

    def _finished_count(self):
        try:
            self._lock.acquire()
            return self.success_parts + self.failed_parts
        finally:
            self._lock.release()

    def wait_all_part_complete(self):
        jobs = self.cur_fragment

        logger.debug("begin wait all part finished. all jobs: %d" % jobs)

        while True:
            gevent.sleep(0.1)
            if self._finished_count() >= jobs:
                self._check_error()
                return

class _SpeedyMultiPartDownloadContext(object):
    iter_chunk_size = 16 * 1024

    def __init__(self, path, conn, tmpdir):
        self.path = path
        self.conn = conn
        self.tmpdir = tmpdir

        self._fragment_tempfiles = []

        self._cur_read_fragment = 0
        self._cursor = 0
        self._max_completed_index = 0
        self._max_completed_byte = 0
        self._max_completed_lock = threading.Lock()
        self._error_lock = threading.Lock()
        self._error = False

        self._last_read_time = time.time()
        self._last_read_lock = threading.Lock()

        self._spawn_downloader()

    def clear(self):
        for f in self._fragment_tempfiles:
            if f:
                f.close()

    def _get_last_read_time(self):
        try:
            self._last_read_lock.acquire()

            return self._last_read_time
        finally:
            self._last_read_lock.release()

    def _update_last_read_time(self):
        try:
            self._last_read_lock.acquire()

            self._last_read_time = time.time()
        finally:
            self._last_read_lock.release()

    def _set_error(self):
        self._error_lock.acquire()
        self._error = True
        self._error_lock.release()

    def _get_error(self):
        try:
            self._error_lock.acquire()
            return self._error
        finally:
            self._error_lock.release()

    def _downloader(self, fragments):
        logger.debug("begin download multi parts.")

        for fragment in fragments:
            fragment_index = fragment['Index']
            fragment_begin = fragment['Start']
            fragment_end = fragment['End']

            try:
                logger.debug("begin download part fragment_index: %d" % fragment_index)
                resp = self.conn.download(self.path, fragment_index, (fragment_begin, fragment_end), stream=False)
                if resp.status_code == 200:
                    content = resp.content
                    self._fragment_tempfiles[fragment_index] = _TempFile(mode='w+b', prefix=self.tmpdir)
                    f = self._fragment_tempfiles[fragment_index].file
                    f.write(content)

                    # seek to 0, ready to be read
                    f.seek(0)
                    self._refresh_max_completed_byte(fragment_index, fragment_end)
                    logger.debug("download part success!!! fragment_index: %d" % fragment_index)

                elif resp.status_code == 404:
                    logger.debug("download part, file not found, path: %s, frament_index: %d" % (self.path, fragment_index))
                    raise exceptions.FileNotFoundError("fetch_part FileNotFound!")
                else:
                    logger.debug("mark else code: %d" % resp.status_code)
                    raise exceptions.UnspecifiedError("unexcept status code: %d" % resp.status_code)
            except:
                logger.error("download part failed, path: %s, index: %d" % (self.path, fragment_index))
                self._set_error()
                return

            # docker read timeout
            now = time.time()
            if now - self._get_last_read_time() > 300:
                logger.debug("speedy long time no read, reader maybe exited!!!")
                self._set_error()
                self.clear()
                return

        logger.debug("end download multi parts.")

    def _spawn_downloader(self):
        fsize, fragments = self.conn.getfileinfo(self.path)
        self._fsize = fsize
        if not fragments:
            return

        self._completed = [0] * len(fragments)
        for i in range(0, len(fragments)):
            self._fragment_tempfiles.append(None)

        gevent.spawn(self._downloader, fragments)

    def _refresh_max_completed_byte(self, fragment_index, fragment_end):
        try:
            self._max_completed_lock.acquire()

            self._completed[fragment_index] = (fragment_index, fragment_end)
            self._max_completed_index = fragment_index
            self._max_completed_byte = fragment_end

        finally:
            self._max_completed_lock.release()

    def _get_max_completed_byte(self):
        try:
            self._max_completed_lock.acquire()
            return self._max_completed_byte
        finally:
            self._max_completed_lock.release()

    def read(self, size):
        if self._cursor >= self._fsize:
            # Read completed and delete all tmpfiles
            self.clear()
            return ''

        # wait data
        if self._cursor + size > self._get_max_completed_byte():
            while self._cursor >= self._get_max_completed_byte():
                gevent.sleep(0.1)
                if self._get_error():
                    break

        if self._get_error():
            self.clear()
            raise RuntimeError("download failed, error flag is on. path: %s" % self.path)

        # update last read time
        logger.debug("speedy update last read time")
        self._update_last_read_time()

        logger.debug("max_completed bytes:%d, cur_fragment:%d" % (self._get_max_completed_byte(), self._cur_read_fragment))

        sz = size
        cur_fragment_info = self._completed[self._cur_read_fragment]
        cur_fragment_left = cur_fragment_info[1] - self._cursor
        if cur_fragment_left == 0:
            # read next fragment file
            self._fragment_tempfiles[self._cur_read_fragment].close()
            self._fragment_tempfiles[self._cur_read_fragment] = None
            self._cur_read_fragment += 1

        elif cur_fragment_left <= sz:
            sz = cur_fragment_left

        f = self._fragment_tempfiles[self._cur_read_fragment].file
        buf = f.read(sz)

        self._cursor += len(buf)
        if not buf:
            message = ('MultiPartDownloadContext; got en empty read on the buffer! '
                       'cursor={0}, size={1}; Transfer interrupted.'.format(
                        self._cursor, self._fsize))
            raise RuntimeError(message)
        return buf

class Storage(driver.Base):
    buffer_size = 1 * 1024 * 1024

    def __init__(self, path=None, config=None):
        self.path = path or ""

        # config tmp file dir
        self.tmpdir = config.tmpdir
        if not self.tmpdir:
            self.tmpdir = "/tmp/"
        logger.debug("tmpdir : %s" % self.tmpdir)

        # config speedy
        speedy_pyclient.init_speedy(config.storage_urls)
        self.speedy_conn = speedy_pyclient.Connection()

        # config speedy fragment size
        fragmentsize_str = config.fragment_size
        if not fragmentsize_str:
            self.fragment_size = 16 * 1024 * 1024
        else:
            if fragmentsize_str[len(fragmentsize_str)-1] == 'M':
                fragmentsize_str = fragmentsize_str[:-1]
            self.fragment_size = int(fragmentsize_str) * 1024 * 1024

        # run speedy heart beat routine
        gevent.spawn(speedy_pyclient.speedy_heart_beater, int(config.heart_beat_interval))

    def get_content(self, path):
        logger.debug("get speedy content path: %s" % path)

        _, fragments = self.speedy_conn.getfileinfo(path)
        if len(fragments) != 1:
            logger.error('little file fragment error! %s' % path)
            raise exceptions.UnspecifiedError("little file invalid fragment info!")

        fragment = fragments[0]

        resp = self.speedy_conn.download(path, 0, (fragment['Start'], fragment['End']))
        if resp.status_code == 200:
            content = resp.content
            logger.debug("get content len: %d" % len(content))
            return content
        elif resp.status_code == 404:
            raise exceptions.FileNotFoundError('%s is not here' % path)
        else:
            logger.error("get content unexcept status code: %d" % resp.status_code)
            raise exceptions.UnspecifiedError("get content unexcept status code: %d" % resp.status_code)

    def put_content(self, path, content):
        logger.debug("put content path: %s" % path)

        bytes_range = (0, len(content))
        resp = self.speedy_conn.upload(path, content, 0, bytes_range, is_last=True)
        if resp.status_code != 200:
            raise exceptions.UnspecifiedError("put speedy content failed: %s, status_code: %d"
                                              % (path, resp.status_code))

    def exists(self, path):
        logger.debug("call exists path: %s" % path)

        return self.speedy_conn.exists(path)

    def remove(self, path):
        logger.debug("remove path: %s" % path)

        resp = self.speedy_conn.delete(path)
        if resp.status_code == 204:
            logger.debug("speedy remove success: %s" % path)
        elif resp.status_code == 404:
            logger.warning("speedy remove, file not found: %s" % path)
            raise exceptions.FileNotFoundError("%s is not here" % path)
        else:
            logger.error("speedy remove, unexcept status code %d" % resp.status_code)
            raise exceptions.UnspecifiedError("speedy remove, unexcept status code %d" % resp.status_code)

    def get_size(self, path):
        logger.debug("get size: %s" % path)

        size, _ = self.speedy_conn.getfileinfo(path)
        return size

    def list_directory(self, path=None):
        logger.debug("list directory path: %s" % path)
        
        resp = self.speedy_conn.list_directory(path)
        if resp.status_code == 200:
            j = resp.json()
            return j["file-list"]
        elif resp.status_code == 404:
            raise exceptions.FileNotFoundError("no such directory: %s" % path)
        else:
            raise exceptions.UnspecifiedError("unexcept status code: %d" % path)

    def stream_read(self, path):
        mc = _SpeedyMultiPartDownloadContext(path, self.speedy_conn, self.tmpdir)
        while True:
            buf = mc.read(self.buffer_size)
            if not buf:
                break
            yield buf

    def stream_write(self, path, fp):
        speedy_mc = _SpeedyMultiPartUploadContext(path, self.speedy_conn, self.fragment_size, self.tmpdir)

        more_data = True
        buf = fp.read(self.buffer_size)
        while True:
            if not buf:
                break
            buf_more = fp.read(self.buffer_size)
            if not buf_more:
                more_data = False

            speedy_mc.push_content(buf, more_data=more_data)

            buf = buf_more

        # wait all part upload completed or failed
        speedy_mc.wait_all_part_complete()

