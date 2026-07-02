import { Client } from 'k6/net/grpc';
import { GRPC_ADDR } from '../config.js';

const clients = {};

/**
 * Return a connected gRPC client for the current VU, using reflection.
 * The client is cached per VU so repeated iterations reuse the connection.
 */
export function getClient() {
  const vu = __VU;
  if (!clients[vu]) {
    const client = new Client();
    client.connect(GRPC_ADDR, { plaintext: true, reflect: true });
    clients[vu] = client;
  }
  return clients[vu];
}

/**
 * Build gRPC metadata with a bearer token.
 */
export function metadata(token) {
  return {
    metadata: {
      authorization: `Bearer ${token}`,
    },
  };
}
