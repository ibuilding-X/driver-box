.PHONY: docker clean docker-no-cache

VERSION=0.3

docker:
	docker build -t driver-box:$(VERSION) .

docker-no-cache:
	docker build -t driver-box:$(VERSION) --no-cache .

clean:
	docker rmi -f driver-box:$(VERSION)
