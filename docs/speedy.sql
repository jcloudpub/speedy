CREATE DATABASE `speedy`;
CREATE DATABASE `metadb`;


--
-- ===================================
-- speedy
-- ===================================
--
USE `speedy`;

--
-- Table structure for table `chunkserver`
--

DROP TABLE IF EXISTS `chunkserver`;
CREATE TABLE `chunkserver` (
  `chunkserver_id` char(32) NOT NULL COMMENT 'id',
  `group_id` smallint(5) unsigned NOT NULL COMMENT 'group id',
  `ip` char(15) NOT NULL COMMENT 'ip',
  `port` int(11) NOT NULL COMMENT 'port',
  `status` int(11) NOT NULL COMMENT 'status 0:INIT 1:RW 2:RO 3: err or death',
  `global_status` int(11) NOT NULL DEFAULT '0' COMMENT 'status 0:untransfer, 8:transfer ',
  `total_free_space` bigint(20) NOT NULL COMMENT 'total free space',
  `max_free_space` bigint(20) NOT NULL COMMENT 'max free space',
  `pend_writes` int(11) NOT NULL COMMENT 'pending queue writes count',
  `writing_count` int(11) NOT NULL COMMENT 'write chunk count',
  `data_path` varchar(255) NOT NULL COMMENT 'data path',
  `reading_count` int(10) unsigned NOT NULL COMMENT 'reading connection',
  `total_chunks` int(10) unsigned NOT NULL COMMENT 'total chunks',
  `conn_counts` int(10) unsigned NOT NULL COMMENT 'connection count',
  `deleted` tinyint(4) NOT NULL DEFAULT '0' COMMENT '0:undeleted, 1:deleted',
  `create_time` timestamp NOT NULL DEFAULT '0000-00-00 00:00:00' COMMENT 'created time',
  `update_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'upated time',
  PRIMARY KEY (`chunkserver_id`),
  UNIQUE KEY `addr` (`ip`,`port`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;


--
-- Table structure for table `gen_fid`
--

DROP TABLE IF EXISTS `gen_fid`;
CREATE TABLE `gen_fid` (
  `fid` bigint(20) unsigned NOT NULL COMMENT 'fid',
  `create_time` timestamp NOT NULL DEFAULT '0000-00-00 00:00:00' COMMENT 'created time',
  `update_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'update time'
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

LOCK TABLES `gen_fid` WRITE;
INSERT INTO `gen_fid` VALUES (1,now(),now());
UNLOCK TABLES;




--
-- ===================================
-- metadb
-- ===================================
--

USE `metadb`;

--
-- Table structure for table `key_list`
--

DROP TABLE IF EXISTS `key_list`;
CREATE TABLE `key_list` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `list_key` varchar(255) DEFAULT NULL,
  `md5_key` char(32) DEFAULT NULL,
  `list_value` varchar(255) DEFAULT NULL,
  `create_time` timestamp NOT NULL DEFAULT '0000-00-00 00:00:00',
  PRIMARY KEY (`id`),
  KEY `md5_key` (`md5_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
