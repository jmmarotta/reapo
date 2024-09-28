.PHONY: install test run clean

install:
	pip install -r requirements.txt

test:
	pytest tests/

run:
	./bin/reapo

clean:
	find . -type f -name '*.pyc' -delete
