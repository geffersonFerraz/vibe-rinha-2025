paymentprocessor: 
	docker build -t geffws/vibe-rinha-2025:latest .
	docker compose down -v --remove-orphans
	docker compose up -d

clean:
	docker compose down -v --remove-orphans

k6:
	k6 run K6/lb.js
