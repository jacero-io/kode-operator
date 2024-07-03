package envoy

#Port: uint32

#SocketAddress: {
    address:    string
    port_value: #Port
}

#Address: {
    socket_address: #SocketAddress
    pipe?: {...}
    envoy_internal_address?: {...}
}
