proxy_ip=
connector_ip=

proxy_external_port=10000
proxy_internal_port=10001
connector_api_port=10002

export UPSTREAM_LISTEN_ADDR="0.0.0.0:${proxy_internal_port}"
export UPSTREAM_API="http://${connector_ip}:${connector_api_port}"
export PROXY_LISTEN_ADDR="0.0.0.0:${proxy_external_port}"

export DOWNSTREAM_ADDR="${proxy_ip}:${proxy_internal_port}"
export CONNECTOR_UPSTREAM_ADDR='localhost:5201'
export API_LISTEN_ADDR="0.0.0.0:${connector_api_port}"
