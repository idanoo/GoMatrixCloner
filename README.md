# gomatrixcloner

- Invite bot to both channels. Success
  
```shell
mkdir data
docker run -it \
  -v data:/data \
  -e "MATRIX_HOST=https://matrix.example" \
  -e "MATRIX_USERNAME=username" \
  -e "MATRIX_PASSWORD=password" \
  -e "MATRIX_SOURCE_ROOM=!example:example.com" \
  -e "MATRIX_DESTINATION_ROOM=!example2:example.com" \
  idanoo/gomatrixcloner:latest
```
