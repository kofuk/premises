export type State = {
  versionName: string;
  serverState?: {
    version: string;
    worldVersionPrev: string;
    worldVersion: string;
    whitelist: string[];
    op: string[];
    serverProps: { [key: string]: string };
  };
};

class Conn {
  public constructor(private conn: Deno.Conn) {}

  public async read(buf: Uint8Array): Promise<number> {
    let total = 0;
    while (total < buf.length) {
      const rdbuf = new Uint8Array(buf.length - total);
      const n = await this.conn.read(rdbuf);
      if (n === null) {
        throw new Error("Read error");
      }
      for (let i = 0; i < n; i++) {
        buf[i + total] = rdbuf[i];
      }
      total += n;
    }
    return total;
  }

  public async write(buf: Uint8Array): Promise<void> {
    const n = await this.conn.write(buf);
    if (n !== buf.length) {
      throw new Error("Write error");
    }
  }
}

const varIntBytes = (value: number): number[] => {
  const buf = [];
  for (;;) {
    if ((value & ~0x7F) == 0) {
      buf.push(value);
      break;
    }
    buf.push((value & 0x7F) | 0x80);
    value >>= 7;
  }
  return buf;
};

const uint16Bytes = (value: number): number[] => {
  const buf = [];
  for (let i = 0; i < 2; i++) {
    buf.push(value >> ((1 - i) * 8) & 0xFF);
  }
  return buf;
};

const uint64Bytes = (value: number): number[] => {
  const buf = [];
  for (let i = 0; i < 8; i++) {
    buf.push(value >> ((7 - i) * 8) & 0xFF);
  }
  return buf;
};

const readVarInt = async (conn: Conn): Promise<number> => {
  let result = 0;
  let pos = 0;
  const buf = new Uint8Array(1);
  for (;;) {
    await conn.read(buf);
    result |= (buf[0] & 0x7F) << pos;
    if ((buf[0] & 0x80) == 0) {
      break;
    }
    pos += 7;
  }
  return result;
};

const readUint64 = async (conn: Conn): Promise<number> => {
  const buf = new Uint8Array(8);
  await conn.read(buf);

  let result = 0;
  for (let i = 0; i < 8; i++) {
    result |= buf[i];
    result <<= 8;
  }

  return result;
};

const sendPacket = async (conn: Conn, packet: {
  id: number;
  data: number[];
}): Promise<void> => {
  const packetIdBuf = varIntBytes(packet.id);
  const lenBuf = varIntBytes(packetIdBuf.length + packet.data.length);

  await conn.write(
    Uint8Array.from([...lenBuf, ...packetIdBuf, ...packet.data]),
  );
};

const receiveHeader = async (
  conn: Conn,
): Promise<{ length: number; id: number }> => {
  const length = await readVarInt(conn);
  const id = await readVarInt(conn);
  return { length, id };
};

const buildHandshakeData = (data: {
  protocolVersion: number;
  serverAddress: string;
  serverPort: number;
  nextState: number;
}): number[] => {
  const serverAddressBuf = [];
  for (let i = 0; i < data.serverAddress.length; i++) {
    serverAddressBuf.push(data.serverAddress.codePointAt(i)!);
  }

  return [
    ...varIntBytes(data.protocolVersion),
    ...varIntBytes(data.serverAddress.length),
    ...serverAddressBuf,
    ...uint16Bytes(data.serverPort),
    ...varIntBytes(data.nextState),
  ];
};

const receiveStatus = async (conn: Conn): Promise<any> => {
  await receiveHeader(conn);
  const len = await readVarInt(conn);
  const buf = new Uint8Array(len);
  await conn.read(buf);
  return JSON.parse(new TextDecoder().decode(buf));
};

export const getState = async (): Promise<State> => {
  const conn = new Conn(
    await Deno.connect({
      port: 25565,
      hostname: "127.0.0.1",
    }),
  );

  // Handshake
  await sendPacket(conn, {
    id: 0,
    data: buildHandshakeData({
      protocolVersion: 765,
      serverAddress: "localhost",
      serverPort: 25565,
      nextState: 1,
    }),
  });

  // Status Request
  await sendPacket(conn, { id: 0, data: [] });

  // Status
  const status = await receiveStatus(conn);

  // Ping
  await sendPacket(conn, { id: 1, data: uint64Bytes(Date.now()) });

  // Pong
  await receiveHeader(conn);
  await readUint64(conn);

  return {
    versionName: status.version.name,
    serverState: status["x-premises"],
  };
};
