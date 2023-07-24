.PHONY: docker clean docker-no-cache

VERSION=0.4

docker:
	docker build -t ibuilding/driver-box:$(VERSION) .

docker-no-cache:
	docker build -t driver-box:$(VERSION) --no-cache .

clean:
	docker rmi -f driver-box:$(VERSION)
