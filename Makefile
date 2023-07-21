.PHONY: docker clean docker-no-cache

VERSION=0.3

docker:
	docker build -t driver-box:$(VERSION) .

docker-no-cache:
	docker build -t driver-box:$(VERSION) --no-cache .

push:
	docker buildx build --no-cache	\
		-t swr.cn-north-4.myhuaweicloud.com/ibuilding/driver-box:$(VERSION) \
		-t swr.cn-north-4.myhuaweicloud.com/ibuilding/driver-box:latest \
		--platform=linux/arm64 . --push

clean:
	docker rmi -f driver-box:$(VERSION)
