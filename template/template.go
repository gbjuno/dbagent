package template

const (
	DOCKER_Single = `net:
  port: {{ .Port }}
operationProfiling: {}
processManagement:
  fork: false
  pidFilePath: /data/mongodb-{{ .Name }}.pid
storage:
  dbPath: /data/mongodb-{{ .Name }}
  engine: wiredTiger
  journal:
     enabled: true
systemLog:
  destination: file
  path: /data/mongodb-{{ .Name }}.log
`

	DOCKER_Replset = `net:
  port: {{ .Port }}
operationProfiling: {}
processManagement:
  fork: false
  pidFilePath: /data/mongodb-{{ .Name }}.pid
replication:
  replSetName: {{ .Name }}
storage:
  dbPath: /data/mongodb-{{ .Name }}
  engine: wiredTiger
  journal:
     enabled: true
systemLog:
  destination: file
  path: /data/mongodb-{{ .Name }}.log
`

	NATIVE_Single = `net:
  port: {{ .Port }}
operationProfiling: {}
processManagement:
  fork: true
  pidFilePath: {{ .DataPath }}/mongodb-{{ .Name }}.pid
storage:
  dbPath: {{ .DataPath }}/mongodb-{{ .Name }}
  engine: wiredTiger
  journal:
     enabled: true
systemLog:
  destination: file
  path: {{ .DataPath }}/mongodb-{{ .Name }}.log
`

	NATIVE_Replset = `net:
  port: {{ .Port }}
operationProfiling: {}
processManagement:
  fork: true
  pidFilePath: {{ .DataPath }}/mongodb-{{ .Name }}.pid
replication:
  replSetName: {{ .Name }}
storage:
  dbPath: {{ .DataPath }}/mongodb-{{ .Name }}
  engine: wiredTiger
  journal:
     enabled: true
systemLog:
  destination: file
  path: {{ .DataPath }}/mongodb-{{ .Name }}.log
`
)
