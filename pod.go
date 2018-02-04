package main

var podDefinition = """
apiVersion: v1
kind: Pod
metadata:
  name: debugger
  labels:
    app: debugger
spec:
  securityContext:
    allowPrivilegeEscalation: true
    privileged: true
  containers:
  - name: debugger
    image: debugger
	ports:
  nodeSelector:
    kubernetes.io/hostname: {{nodeName}}
"""