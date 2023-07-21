.PHONY: docker clean docker-no-cache

VERSION=0.4

docker:
	docker build \
		-t swr.cn-north-4.myhuaweicloud.com/ibuilding/driver-box:$(VERSION) \
        -t swr.cn-north-4.myhuaweicloud.com/ibuilding/driver-box:latest \
		-t ibuilding/driver-box:$(VERSION) .

docker-no-cache:
	docker build -t driver-box:$(VERSION) --no-cache .

clean:
	docker rmi -f driver-box:$(VERSION)
