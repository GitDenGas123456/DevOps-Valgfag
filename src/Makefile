.PHONY: init run test

PYTHON = backend/venv/bin/python

init:
	PYTHONPATH=backend $(PYTHON) -c "from app import init_db; init_db()"

run:
	$(PYTHON) backend/app.py

test:
	PYTHONPATH=backend $(PYTHON) backend/app_tests.py
