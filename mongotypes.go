package main

import (
	"time"
)

type Asserts struct {
	Regular   int `json:"regular,omitempty"`
	Warning   int `json:"warning,omitempty"`
	Msg       int `json:"msg,omitempty"`
	User      int `json:"user,omitempty"`
	Rollovers int `json:"rollovers,omitempty"`
}

type BackgroundFlushing struct {
	Flushes       int64 `json:"flushes,omitempty"`
	Total_ms      int64 `json:"total_ms,omitempty"`
	Average_ms    int64 `json:"average_ms,omitempty"`
	Last_ms       int64 `json:"last_ms,omitempty"`
	Last_finished int64 `json:"last_finished,omitempty"`
}

type Connections struct {
	Current      int `json:"current,omitempty"`
	Available    int `json:"available,omitempty"`
	TotalCreated int `json:"totalCreated,omitempty"`
}

type TimeMs struct {
	Dt                 int64 `json:"Dt,omitempty"`
	PrepLogBuffer      int64 `json:"prepLogBuffer,omitempty"`
	WriteToJournal     int64 `json:"writeToJournal,omitempty"`
	WriteToDataFiles   int64 `json:"writeToDataFiles,omitempty"`
	RemapPrivateView   int64 `json:"remapPrivateView,omitempty"`
	Commits            int64 `json:"commits,omitempty"`
	CommitsInWriteLock int64 `json:"commitsInWriteLock,omitempty"`
}

type Dur struct {
	Commits            int64  `json:"commits,omitempty"`
	JournaledMB        int64  `json:"journaledMB,omitempty"`
	WriteToDataFilesMB int64  `json:"writeToDataFilesMB,omitempty"`
	Compression        int64  `json:"compression,omitempty"`
	CommitsInWriteLock int64  `json:"commitsInWriteLock,omitempty"`
	EarlyCommits       int64  `json:"earlyCommits,omitempty"`
	DTimeMs            TimeMs `json:"timeMs,omitempty"`
}

type Extra_info struct {
	Note             string `json:"note,omitempty"`
	Heap_usage_bytes int64  `json:"heap_usage_bytes,omitempty"`
	Page_faults      int64  `json:"page_faults,omitempty"`
}

type GLock struct {
	Total   int64 `json:"total,omitempty"`
	Readers int64 `json:"readers,omitempty"`
	Writers int64 `json:"writers,omitempty"`
}

type GlobalLock struct {
	TotalTime    int64 `json:"totalTime,omitempty"`
	CurrentQueue GLock `json:"currentQueue,omitempty"`
	ActiveQueue  GLock `json:"activeQueue,omitempty"`
}

type Lock struct {
	SR int64 `json:"r,omitempty"`
	SW int64 `json:"w,omitempty"`
	BR int64 `json:"R,omitempty"`
	BW int64 `json:"W,omitempty"`
}

type locks struct {
	GlobalLock map[string]Lock `json:"Global,omitempty"`
	Database   map[string]Lock `json:"Database,omitempty"`
	Collection map[string]Lock `json:"Collection,omitempty"`
	Metadata   map[string]Lock `json:"MetaData,omitempty"`
	Oplog      map[string]Lock `json:"oplog,omitempty"`
}

type Network struct {
	BytesIn     int64 `json:"bytesIn,omitempty"`
	BytesOut    int64 `json:"bytesOut,omitempty"`
	NumRequests int64 `json:"numRequests,omitempty"`
}

type OpLatencies struct {
	Reads    int64 `json:"reads,omitempty"`
	Writes   int64 `json:"writes,omitempty"`
	Commands int64 `json:"commands,omitempty"`
}

type Opcounters struct {
	Insert  int `json:"insert,omitempty"`
	Query   int `json:"query,omitempty"`
	Update  int `json:"update,omitempty"`
	Delete  int `json:"delete,omitempty"`
	Getmore int `json:"getmore,omitempty"`
	Command int `json:"command,omitempty"`
}

type OpcountersRepl struct {
	Insert  int64 `json:"insert,omitempty"`
	Query   int64 `json:"query,omitempty"`
	Update  int64 `json:"update,omitempty"`
	Delete  int64 `json:"delete,omitempty"`
	Getmore int64 `json:"getmore,omitempty"`
	Command int64 `json:"command,omitempty"`
}

type LastDeleteStats struct {
	DeletedDocs      int64 `json:"deletedDocs,omitempty"`
	QueueStart       int64 `json:"queueStart,omitempty"`
	QueueEnd         int64 `json:"queueEnd,omitempty"`
	DeleteStart      int64 `json:"deleteStart,omitempty"`
	DeleteEnd        int64 `json:"deleteEnd,omitempty"`
	WaitForReplStart int64 `json:"waitForReplStart,omitempty"`
	WaitForReplEnd   int64 `json:"waitForReplEnd,omitempty"`
}

type RangeDeleter struct {
	lastDeleteStatsList []LastDeleteStats `json:"lastDeleteStats,omitempty"`
}

type Security struct {
	SSLServerSubjectName               string    `json:"SSLServerSubjectName,omitempty"`
	SSLServerHasCertificateAuthority   bool      `json:"SSLServerHasCertificateAuthority,omitempty"`
	SSLServerCertificateExpirationDate time.Time `json:"SSLServerCertificateExpirationDate,omitempty"`
}

type StorageEngine struct {
	Name                   string `json:"name,omitempty"`
	SupportsCommittedReads bool   `json:"supportsCommittedReads,omitempty"`
	Persistent             bool   `json:"persistent,omitempty"`
}

type Mem struct {
	Bits              int  `json:"bits,omitempty"`
	Resident          int  `json:"resident,omitempty"`
	Virtual           int  `json:"virtual,omitempty"`
	Supported         bool `json:"suppported,omitempty"`
	Mapped            int  `json:"mapped,omitempty"`
	MappedWithJournal int  `json:"mappedWithJournal,omitempty"`
}
