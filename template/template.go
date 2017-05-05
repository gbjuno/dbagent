package template

const (
	Single = `net:
  port: {{ .Port }}
operationProfiling: {}
processManagement:
  fork: "false"
storage:
  dbPath: /data
  engine: wiredTiger
systemLog:
  destination: file
  path: /data/mongodb.log
`
	Replset = `net:
  port: {{ .Port }}
operationProfiling: {}
processManagement:
  fork: "false"
replication:
  replSetName: {{ .Name }}
storage:
  dbPath: /data
  engine: wiredTiger
systemLog:
  destination: file
  path: /data/mongodb.log
`
)
