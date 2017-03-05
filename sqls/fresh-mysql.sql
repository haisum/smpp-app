-- --------------------------------------------------------
-- Host:                         localhost
-- Server version:               5.7.16 - MySQL Community Server (GPL)
-- Server OS:                    Win64
-- HeidiSQL Version:             9.4.0.5125
-- --------------------------------------------------------

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET NAMES utf8 */;
/*!50503 SET NAMES utf8mb4 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;

-- Dumping structure for table hsmppdb.campaign
CREATE TABLE IF NOT EXISTS `campaign` (
  `ID` int(11) NOT NULL AUTO_INCREMENT,
  `SendAfter` varchar(50) NOT NULL DEFAULT '',
  `SendBefore` varchar(50) NOT NULL DEFAULT '',
  `Dst` varchar(50) NOT NULL DEFAULT '',
  `Priority` int(11) NOT NULL DEFAULT '1',
  `ScheduledAt` bigint(20) NOT NULL DEFAULT '0',
  `SubmittedAt` bigint(20) NOT NULL DEFAULT '0',
  `Msg` text NOT NULL,
  `Description` text NOT NULL,
  `Src` varchar(50) NOT NULL,
  `Username` varchar(100) NOT NULL,
  `NumFileID` int(11) NOT NULL,
  PRIMARY KEY (`ID`),
  KEY `SubmittedAt` (`SubmittedAt`),
  KEY `Src` (`Src`),
  KEY `Username` (`Username`),
  KEY `campaign_numfileid` (`NumFileID`),
  CONSTRAINT `campaign_numfileid` FOREIGN KEY (`NumFileID`) REFERENCES `numfile` (`ID`),
  CONSTRAINT `campaign_username` FOREIGN KEY (`Username`) REFERENCES `user` (`Username`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8 ROW_FORMAT=DYNAMIC;

-- Dumping data for table hsmppdb.campaign: ~0 rows (approximately)
/*!40000 ALTER TABLE `campaign` DISABLE KEYS */;
/*!40000 ALTER TABLE `campaign` ENABLE KEYS */;

-- Dumping structure for table hsmppdb.message
CREATE TABLE IF NOT EXISTS `message` (
  `ID` int(11) NOT NULL AUTO_INCREMENT,
  `Enc` varchar(50) NOT NULL DEFAULT 'latin',
  `SendAfter` varchar(50) NOT NULL DEFAULT '',
  `SendBefore` varchar(50) NOT NULL DEFAULT '',
  `Dst` varchar(50) NOT NULL DEFAULT '',
  `RespID` varchar(50) NOT NULL DEFAULT '',
  `Priority` int(11) NOT NULL DEFAULT '1',
  `ScheduledAt` bigint(20) NOT NULL DEFAULT '0',
  `ConnectionGroup` varchar(50) NOT NULL DEFAULT 'Default',
  `Msg` text NOT NULL,
  `SentAt` bigint(20) NOT NULL DEFAULT '0',
  `Status` varchar(50) NOT NULL,
  `DeliveredAt` bigint(20) NOT NULL DEFAULT '0',
  `Src` varchar(50) NOT NULL,
  `Fields` json DEFAULT NULL,
  `QueuedAt` bigint(20) NOT NULL DEFAULT '0',
  `Connection` varchar(50) NOT NULL,
  `Error` varchar(50) NOT NULL DEFAULT '',
  `Username` varchar(50) NOT NULL,
  `IsFlash` tinyint(4) NOT NULL DEFAULT '0',
  `RealMsg` varchar(50) NOT NULL DEFAULT '',
  `CampaignID` int(11) NOT NULL DEFAULT '1',
  `Campaign` varchar(50) NOT NULL DEFAULT '',
  `DeliverySM` json DEFAULT NULL,
  `Total` int(11) NOT NULL DEFAULT '1',
  PRIMARY KEY (`ID`),
  KEY `Dst` (`Dst`),
  KEY `RespID` (`RespID`),
  KEY `ConnectionGroup` (`ConnectionGroup`),
  KEY `Src` (`Src`),
  KEY `Connection` (`Connection`),
  KEY `Username` (`Username`),
  KEY `Message_CampaignID` (`CampaignID`),
  CONSTRAINT `Message_CampaignID` FOREIGN KEY (`CampaignID`) REFERENCES `campaign` (`ID`),
  CONSTRAINT `Message_Username` FOREIGN KEY (`Username`) REFERENCES `user` (`Username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

-- Dumping data for table hsmppdb.message: ~0 rows (approximately)
/*!40000 ALTER TABLE `message` DISABLE KEYS */;
/*!40000 ALTER TABLE `message` ENABLE KEYS */;

-- Dumping structure for table hsmppdb.numfile
 CREATE TABLE IF NOT EXISTS `numfile` (
  `ID` int(11) NOT NULL AUTO_INCREMENT,
  `Name` varchar(200) NOT NULL,
  `Description` text NOT NULL,
  `LocalName` varchar(200) NOT NULL,
  `Username` varchar(100) NOT NULL,
  `SubmittedAt` bigint(20) NOT NULL DEFAULT '0',
  `Deleted` tinyint(4) NOT NULL DEFAULT '0',
  `Type` varchar(25) NOT NULL,
  PRIMARY KEY (`ID`),
  KEY `Username` (`Username`),
  KEY `Deleted` (`Deleted`),
  KEY `Username_Deleted` (`Username`,`Deleted`),
  CONSTRAINT `numfile_username` FOREIGN KEY (`Username`) REFERENCES `user` (`Username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

-- Dumping data for table hsmppdb.numfile: ~0 rows (approximately)
/*!40000 ALTER TABLE `numfile` DISABLE KEYS */;
/*!40000 ALTER TABLE `numfile` ENABLE KEYS */;

-- Dumping structure for table hsmppdb.settings
CREATE TABLE IF NOT EXISTS `settings` (
  `ID` int(11) NOT NULL AUTO_INCREMENT,
  `Name` varchar(50) NOT NULL,
  `Value` json NOT NULL,
  PRIMARY KEY (`ID`),
  UNIQUE KEY `Name` (`Name`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8;

-- Dumping data for table hsmppdb.settings: ~1 rows (approximately)
/*!40000 ALTER TABLE `settings` DISABLE KEYS */;
INSERT INTO `settings` (`ID`, `Name`, `Value`) VALUES
  (1, 'config', '{"ConnGroups": [{"Name": "Default", "Conns": [{"ID": "du-1", "URL": "192.168.0.105:2775", "Pfxs": ["+97105", "+97106"], "Size": 5, "Time": 1, "User": "smppclient1", "Fields": {"ESMClass": 0, "ProtocolID": 0, "DestAddrNPI": 0, "DestAddrTON": 0, "ServiceType": "", "PriorityFlag": 0, "SourceAddrNPI": 0, "SourceAddrTON": 0, "SMDefaultMsgID": 0, "ReplaceIfPresentFlag": 0, "ScheduleDeliveryTime": ""}, "Passwd": "password", "Receiver": ""}, {"ID": "du-2", "URL": "192.168.0.105:2775", "Pfxs": ["+97107", "+97108"], "Size": 5, "Time": 1, "User": "smppclient2", "Passwd": "password", "Receiver": ""}], "DefaultPfx": "+97105"}, {"Name": "AADC", "Conns": [{"ID": "du-2", "URL": "192.168.0.105:2775", "Pfxs": ["+97107", "+97108"], "Size": 5, "Time": 1, "User": "smppclient2", "Passwd": "password", "Receiver": ""}], "DefaultPfx": "+97105"}]}');
/*!40000 ALTER TABLE `settings` ENABLE KEYS */;

-- Dumping structure for table hsmppdb.token
CREATE TABLE IF NOT EXISTS `token` (
  `ID` int(11) NOT NULL AUTO_INCREMENT,
  `Token` varchar(100) NOT NULL,
  `Username` varchar(100) NOT NULL,
  `Validity` bigint(20) NOT NULL DEFAULT '0',
  `LastAccessed` bigint(20) NOT NULL DEFAULT '0',
  PRIMARY KEY (`ID`),
  KEY `Token` (`Token`),
  KEY `Username` (`Username`),
  CONSTRAINT `token_username` FOREIGN KEY (`Username`) REFERENCES `user` (`Username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

-- Dumping data for table hsmppdb.token: ~0 rows (approximately)
/*!40000 ALTER TABLE `token` DISABLE KEYS */;
/*!40000 ALTER TABLE `token` ENABLE KEYS */;

-- Dumping structure for table hsmppdb.user
CREATE TABLE IF NOT EXISTS `user` (
  `ID` int(11) NOT NULL AUTO_INCREMENT,
  `Username` varchar(100) NOT NULL,
  `Password` varchar(100) NOT NULL,
  `Name` varchar(100) NOT NULL,
  `Email` varchar(100) NOT NULL,
  `ConnectionGroup` varchar(100) NOT NULL,
  `RegisteredAt` bigint(20) NOT NULL DEFAULT '0',
  `Permissions` json NOT NULL,
  PRIMARY KEY (`ID`),
  KEY `Username` (`Username`),
  KEY `ConnectionGroup` (`ConnectionGroup`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8;

-- Dumping data for table hsmppdb.user: ~1 rows (approximately)
/*!40000 ALTER TABLE `user` DISABLE KEYS */;
INSERT INTO `user` (`ID`, `Username`, `Password`, `Name`, `Email`, `ConnectionGroup`, `RegisteredAt`, `Permissions`) VALUES
  (1, 'admin', '$2a$10$2dgWOU4i12GnSyKl2JfpT.IYWNSaE0vXp2IJvtTLRFUjrs4qQXJre', 'Admin', 'admin@localhost', 'Default', 0, '["Add users", "Edit users", "List users", "Show config", "Edit config", "Send message", "Start a campaign", "List messages", "List number files", "Delete a number file", "List campaigns", "Stop campaign", "Retry campaign", "Get status of services", "Mask Messages"]');
/*!40000 ALTER TABLE `user` ENABLE KEYS */;

/*!40101 SET SQL_MODE=IFNULL(@OLD_SQL_MODE, '') */;
/*!40014 SET FOREIGN_KEY_CHECKS=IF(@OLD_FOREIGN_KEY_CHECKS IS NULL, 1, @OLD_FOREIGN_KEY_CHECKS) */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
