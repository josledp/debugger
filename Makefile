all:

container: container_build container_upload

container_build:
	docker build -t josledp/debugpod container/

container_upload:
	 DOCKER_ID_USER="josledp" docker login
	 docker push josledp/debugpod
	 rm $(HOME)/.docker/config.json
