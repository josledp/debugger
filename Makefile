all:

container: container_build container_upload

container_build:
	docker build -t josledp/debugger container/

container_upload:
	 DOCKER_ID_USER="josledp" docker login
	 docker push josledp/debugger
