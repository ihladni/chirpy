# Čitanje

https://www.rfc-editor.org/rfc/rfc9110.html

# Primjeri

## Na početku, najbolje reset stanja napravit

POST http://localhost:8080/admin/reset

## Dohvat svih poruka

GET http://localhost:8080/api/chirps

## Dohvat usera

POST http://localhost:8080/api/users
content-type: application/json

{
  "email": "saul@bettercall.com"
}

## Slanje nove poruke (poruka 1)
UserID uzeti iz "Dohvat usera".

POST http://localhost:8080/api/chirps
content-type: application/json

{
  "body": "If you're committed enough, you can make any story work.",
  "user_id": "010f572f-c3e8-4674-b2c3-634aa41c2df2"
}

## Slanje nove poruke (poruka 2)

POST http://localhost:8080/api/chirps
content-type: application/json

{
  "body": "I once told a woman I was Kevin Costner, and it worked because I believed it.",
  "user_id": "db2b771b-fc7e-469a-9a1d-fb3a2d19b1a7"
}

# Dohvat poruke preko ID-ja
Treba uzeti neki ID preko "Dohvat svih poruka".

GET http://localhost:8080/api/chirps/e51d9dec-c666-4366-bf18-2b4b68d78309

# Dohvat poruke preko ID-ja (poruka ne postoji, 404 expected)
Treba uzeti neki ID preko "Dohvat svih poruka".

GET http://localhost:8080/api/chirps/e51d9dec-c666-4366-bf10-2b4b68d78309

