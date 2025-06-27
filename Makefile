.PHONY: start

start:
	# install frontend dependencies
	cd frontend && npm init -y && npm install

	# run server
	cd backend && go run .

