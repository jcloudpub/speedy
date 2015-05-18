-- default database "speedy"
DROP DATABASE IF EXISTS `speedy`;
CREATE DATABASE `speedy`;

DROP TABLE IF EXISTS `speedy`.`chunkserver`;
CREATE TABLE `speedy`.`chunkserver` (
  `chunkserver_id` 		char(32) 	NOT NULL COMMENT 'id',
  `group_id` 			smallint(5) unsigned NOT NULL COMMENT 'group id',
  `ip`					char(15) NOT NULL COMMENT 'ip',
  `port` 				int(11) NOT NULL COMMENT 'port',
  `status` 				int(11) NOT NULL COMMENT 'status 0:RW 1:abort 2: err or death',
  `global_status` 		int(11) NOT NULL DEFAULT 0 COMMENT 'status 0:untransfer, 8:transfer ',
  `abnormal_count` 		int(11) NOT NULL DEFAULT '0',
  `total_free_space` 	bigint(20) NOT NULL COMMENT 'total free space',
  `max_free_space` 		bigint(20) NOT NULL COMMENT 'max free space',
  `pend_writes` 		int(11) NOT NULL COMMENT 'pending queue writes count',
  `writing_count` 		int(11) NOT NULL COMMENT 'write chunk count',
  `data_path` 			varchar(255) NOT NULL COMMENT 'data path',
  `reading_count` 		int(10) unsigned NOT NULL COMMENT 'reading connection',
  `total_chunks` 		int(10) unsigned NOT NULL COMMENT 'total chunks',
  `conn_counts` 		int(10) unsigned NOT NULL COMMENT 'connection count',
  `deleted` 			tinyint(4) NOT NULL DEFAULT '0' COMMENT '0:undeleted, 1:deleted',
  `create_time` 		timestamp NOT NULL DEFAULT '0000-00-00 00:00:00' COMMENT 'created time',
  `update_time` 		timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'upated time',
  PRIMARY KEY (`chunkserver_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

DROP TABLE IF EXISTS `speedy`.`gen_fid`;
CREATE TABLE `speedy`.`gen_fid` (
  `fid`					bigint(20) unsigned NOT NULL COMMENT 'fid',
  `create_time`			timestamp NOT NULL DEFAULT '0000-00-00 00:00:00' COMMENT 'created time',
  `update_time`			timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'update time'
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

--init gen_fid table
INSERT INTO `speedy`.`gen_fid` (`fid`, `create_time`) VALUES (1, now());
