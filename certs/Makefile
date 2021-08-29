
COUNTRY := CA
STATE := ON
COMPANY := NONE
CN_SERVER := server
CN_CLIENT1 := alice
CN_CLIENT2 := bob

all: keys


keys:
	openssl genrsa -out rootCA.key 2048
	openssl genrsa -out client1.key 2048
	openssl genrsa -out client2.key 2048
	openssl genrsa -out server.key 2048

# create and self sign root certificate
	echo -ne "$(COUNTRY)\n$(STATE)\n\n$(COMPANY)\n\n\n\n" | openssl req -x509 -new -nodes -key rootCA.key -sha256 -days 360 -out rootCA.crt

# server and client
	openssl genrsa -out server.key 2048
	openssl genrsa -out client1.key 2048
	openssl genrsa -out client2.key 2048

# create config files for server and client
	sh mkconf.sh server > server.conf
#sh mkconf.sh alice > client1.conf
#sh mkconf.sh bob > client2.conf

# generate csr files for server and client
	openssl req -new -sha256 -key server.key -subj "/C=$(COUNTRY)/ST=$(STATE)/O=$(COMPANY)/CN=$(CN_SERVER)" \
		-reqexts SAN \
		-config server.conf \
		-out server.csr

	openssl req -new -sha256 -key client1.key -subj "/C=$(COUNTRY)/ST=$(STATE)/O=$(COMPANY)/CN=$(CN_CLIENT1)" \
		-out client1.csr

	openssl req -new -sha256 -key client2.key -subj "/C=$(COUNTRY)/ST=$(STATE)/O=$(COMPANY)/CN=$(CN_CLIENT2)" \
		-out client2.csr

	sh mksan.sh server $(COUNTRY) $(STATE) "$(COMPANY)" > server.san.conf
#	sh mksan.sh client $(COUNTRY) $(STATE) "$(COMPANY)" > client.san.conf

# create certificates
	openssl x509 -req -in server.csr -CA ./rootCA.crt -CAcreateserial -CAkey ./rootCA.key \
		-out server.crt -days 360 -sha256 \
		-extfile server.san.conf -extensions req_ext

	openssl x509 -req -in client1.csr -CA ./rootCA.crt -CAkey ./rootCA.key \
		-out client1.crt -days 360 -sha256

	openssl x509 -req -in client2.csr -CA ./rootCA.crt -CAkey ./rootCA.key \
		-out client2.crt -days 360 -sha256

	openssl x509 -in server.crt -text -noout
	openssl x509 -in client1.crt -text -noout
	openssl x509 -in client2.crt -text -noout


.PHONY: clean
clean:
	rm -f *.key *.crt *.csr *.conf
