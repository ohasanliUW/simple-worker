
COUNTRY := CA
STATE := ON
COMPANY := NONE
CN_SERVER := server
CN_CLIENT := alice

all:
	openssl genrsa -out rootCA.key 2048

# create and self sign root certificate
	echo -ne "$(COUNTRY)\n$(STATE)\n\n$(COMPANY)\n\n\n\n" | openssl req -x509 -new -nodes -key rootCA.key -sha256 -days 360 -out rootCA.crt

# server and client
	openssl genrsa -out server.key 2048
	openssl genrsa -out client.key 2048

# create config files for server and client
	sh mkconf.sh localhost > server.conf

# generate csr files for server and client
	openssl req -new -sha256 -key server.key -subj "/C=$(COUNTRY)/ST=$(STATE)/O=$(COMPANY)/CN=$(CN_SERVER)" \
		-reqexts SAN \
		-config server.conf \
		-out server.csr

	openssl req -new -sha256 -key client.key -subj "/C=$(COUNTRY)/ST=$(STATE)/O=$(COMPANY)/CN=$(CN_CLIENT)" \
		-out client.csr

	sh mksan.sh localhost $(COUNTRY) $(STATE) "$(COMPANY)" > server.san.conf

# create certificates
	openssl x509 -req -in server.csr -CA ./rootCA.crt -CAcreateserial -CAkey ./rootCA.key \
		-out server.crt -days 360 -sha256 \
		-extfile server.san.conf -extensions req_ext

	openssl x509 -req -in client.csr -CA ./rootCA.crt -CAcreateserial -CAkey ./rootCA.key \
		-out client.crt -days 360 -sha256

	openssl x509 -in server.crt -text -noout
	openssl x509 -in client.crt -text -noout


#	sudo cp rootCA.crt /usr/local/share/ca-certificates/
#	sudo update-ca-certificates
.PHONY: clean
clean:
	rm -f *.key *.crt *.csr *.conf
