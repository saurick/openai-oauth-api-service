.PHONY: build clean rebuild install start

clean:
	rm -rf build build.debug.log

build: clean
	pnpm build --debug > build.debug.log 2>&1
	tail -n 80 build.debug.log

rebuild: clean build

install:
	pnpm install

start:
	pnpm start