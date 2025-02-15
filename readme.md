# cURL testiranje
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{"body":"Hello world"}' \
  http://localhost:8080/api/validate_chirp
