.PHONY: docker clean

VERSION=0.0.1

docker:
	docker build -t driver-box:$(VERSION) .

docker-no-cache:
	docker build -t driver-box:$(VERSION) --no-cache .

clean:
	docker rmi -f driver-box:$(VERSION)
