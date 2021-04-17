// Gamcro – Game Macros

package main

const (
	docSrvAddrFlag = `This is the local address the API server and the UI server
listen to. Most people should be fine with the default. If
you just need to set a different port number <port> set
the flag value ":<port>". For more details read the description
of the address parameter in https://golang.org/pkg/net/#Listen
`

	docTlsCertFlag = `TLS certificate file to use for HTTPS.
If neither the certificate nor the key file exist, Gamcro will
generate them with an self-signed X.509 certificate.
`

	docTlsKeyFlag = `TLS key file to use for HTTPS.
If neither the certificate nor the key file exist, Gamcro will
generate them with an self-signed X.509 certificate.
`

	docAuthCredsFlag = `Access to the API server is protected by HTTP basic auth.
A single <user>:<password> pair will be used to check a user's
authorization. Use this flags to set the <user>:<password>
credentials. The current settings are determined like this:
 - When the 'auth' falg is empty, Gamcro checks for the file
   '%[1]s' in the same folder as the Gamcro executable.
   When present Gamcro reads <user>:<password> from the first
   text line of the file. Keep read access to that file as
   restrictive as possible.
 - When 'auth' flag is set to ":" Gamcro ignores the '%[1]s'
   file in the executables directory and reads <user> and
   <password> from the terminal.
 - Otherwise when 'auth' flag's value contains ':' Gamcro
   considers 'auth' flag to be <user>:<password> and uses is.
 - Else 'auth' flag is considered to be a filename and Gamcro
   will try to read <user>:<password> from the first text line
   of that file.`

	docTxtLimitFlag = `Limit the length of text input to API.`
)
