// node_modules/@connectrpc/connect/dist/esm/code.js
var Code;
(function(Code2) {
  Code2[Code2["Canceled"] = 1] = "Canceled";
  Code2[Code2["Unknown"] = 2] = "Unknown";
  Code2[Code2["InvalidArgument"] = 3] = "InvalidArgument";
  Code2[Code2["DeadlineExceeded"] = 4] = "DeadlineExceeded";
  Code2[Code2["NotFound"] = 5] = "NotFound";
  Code2[Code2["AlreadyExists"] = 6] = "AlreadyExists";
  Code2[Code2["PermissionDenied"] = 7] = "PermissionDenied";
  Code2[Code2["ResourceExhausted"] = 8] = "ResourceExhausted";
  Code2[Code2["FailedPrecondition"] = 9] = "FailedPrecondition";
  Code2[Code2["Aborted"] = 10] = "Aborted";
  Code2[Code2["OutOfRange"] = 11] = "OutOfRange";
  Code2[Code2["Unimplemented"] = 12] = "Unimplemented";
  Code2[Code2["Internal"] = 13] = "Internal";
  Code2[Code2["Unavailable"] = 14] = "Unavailable";
  Code2[Code2["DataLoss"] = 15] = "DataLoss";
  Code2[Code2["Unauthenticated"] = 16] = "Unauthenticated";
})(Code || (Code = {}));

// node_modules/@connectrpc/connect/dist/esm/protocol-connect/code-string.js
function codeToString(value) {
  const name = Code[value];
  if (typeof name != "string") {
    return value.toString();
  }
  return name[0].toLowerCase() + name.substring(1).replace(/[A-Z]/g, (c) => "_" + c.toLowerCase());
}
function codeFromString(value) {
  if (!stringToCode) {
    stringToCode = {};
    for (const value2 of Object.values(Code)) {
      if (typeof value2 == "string") {
        continue;
      }
      stringToCode[codeToString(value2)] = value2;
    }
  }
  return stringToCode[value];
}
var stringToCode;

// node_modules/@connectrpc/connect/dist/esm/connect-error.js
var createMessage = function(message, code3) {
  return message.length ? `[${codeToString(code3)}] ${message}` : `[${codeToString(code3)}]`;
};

class ConnectError extends Error {
  constructor(message, code3 = Code.Unknown, metadata, outgoingDetails, cause) {
    super(createMessage(message, code3));
    this.name = "ConnectError";
    Object.setPrototypeOf(this, new.target.prototype);
    this.rawMessage = message;
    this.code = code3;
    this.metadata = new Headers(metadata !== null && metadata !== undefined ? metadata : {});
    this.details = outgoingDetails !== null && outgoingDetails !== undefined ? outgoingDetails : [];
    this.cause = cause;
  }
  static from(reason, code3 = Code.Unknown) {
    if (reason instanceof ConnectError) {
      return reason;
    }
    if (reason instanceof Error) {
      if (reason.name == "AbortError") {
        return new ConnectError(reason.message, Code.Canceled);
      }
      return new ConnectError(reason.message, code3, undefined, undefined, reason);
    }
    return new ConnectError(String(reason), code3, undefined, undefined, reason);
  }
  static [Symbol.hasInstance](v) {
    if (!(v instanceof Error)) {
      return false;
    }
    if (Object.getPrototypeOf(v) === ConnectError.prototype) {
      return true;
    }
    return v.name === "ConnectError" && "code" in v && typeof v.code === "number" && "metadata" in v && "details" in v && Array.isArray(v.details) && "rawMessage" in v && typeof v.rawMessage == "string" && "cause" in v;
  }
  findDetails(typeOrRegistry) {
    const registry = "typeName" in typeOrRegistry ? {
      findMessage: (typeName) => typeName === typeOrRegistry.typeName ? typeOrRegistry : undefined
    } : typeOrRegistry;
    const details = [];
    for (const data of this.details) {
      if ("getType" in data) {
        if (registry.findMessage(data.getType().typeName)) {
          details.push(data);
        }
        continue;
      }
      const type = registry.findMessage(data.type);
      if (type) {
        try {
          details.push(type.fromBinary(data.value));
        } catch (_) {
        }
      }
    }
    return details;
  }
}

// node_modules/@bufbuild/protobuf/dist/esm/private/assert.js
function assert(condition, msg) {
  if (!condition) {
    throw new Error(msg);
  }
}
function assertInt32(arg) {
  if (typeof arg !== "number")
    throw new Error("invalid int 32: " + typeof arg);
  if (!Number.isInteger(arg) || arg > INT32_MAX || arg < INT32_MIN)
    throw new Error("invalid int 32: " + arg);
}
function assertUInt32(arg) {
  if (typeof arg !== "number")
    throw new Error("invalid uint 32: " + typeof arg);
  if (!Number.isInteger(arg) || arg > UINT32_MAX || arg < 0)
    throw new Error("invalid uint 32: " + arg);
}
function assertFloat32(arg) {
  if (typeof arg !== "number")
    throw new Error("invalid float 32: " + typeof arg);
  if (!Number.isFinite(arg))
    return;
  if (arg > FLOAT32_MAX || arg < FLOAT32_MIN)
    throw new Error("invalid float 32: " + arg);
}
var FLOAT32_MAX = 340282346638528860000000000000000000000;
var FLOAT32_MIN = -340282346638528860000000000000000000000;
var UINT32_MAX = 4294967295;
var INT32_MAX = 2147483647;
var INT32_MIN = -2147483648;

// node_modules/@bufbuild/protobuf/dist/esm/private/enum.js
function getEnumType(enumObject) {
  const t = enumObject[enumTypeSymbol];
  assert(t, "missing enum type on enum object");
  return t;
}
function setEnumType(enumObject, typeName, values, opt) {
  enumObject[enumTypeSymbol] = makeEnumType(typeName, values.map((v) => ({
    no: v.no,
    name: v.name,
    localName: enumObject[v.no]
  })), opt);
}
function makeEnumType(typeName, values, _opt) {
  const names = Object.create(null);
  const numbers = Object.create(null);
  const normalValues = [];
  for (const value of values) {
    const n = normalizeEnumValue(value);
    normalValues.push(n);
    names[value.name] = n;
    numbers[value.no] = n;
  }
  return {
    typeName,
    values: normalValues,
    findName(name) {
      return names[name];
    },
    findNumber(no) {
      return numbers[no];
    }
  };
}
function makeEnum(typeName, values, opt) {
  const enumObject = {};
  for (const value of values) {
    const n = normalizeEnumValue(value);
    enumObject[n.localName] = n.no;
    enumObject[n.no] = n.localName;
  }
  setEnumType(enumObject, typeName, values, opt);
  return enumObject;
}
var normalizeEnumValue = function(value) {
  if ("localName" in value) {
    return value;
  }
  return Object.assign(Object.assign({}, value), { localName: value.name });
};
var enumTypeSymbol = Symbol("@bufbuild/protobuf/enum-type");

// node_modules/@bufbuild/protobuf/dist/esm/message.js
class Message {
  equals(other) {
    return this.getType().runtime.util.equals(this.getType(), this, other);
  }
  clone() {
    return this.getType().runtime.util.clone(this);
  }
  fromBinary(bytes, options) {
    const type = this.getType(), format = type.runtime.bin, opt = format.makeReadOptions(options);
    format.readMessage(this, opt.readerFactory(bytes), bytes.byteLength, opt);
    return this;
  }
  fromJson(jsonValue, options) {
    const type = this.getType(), format = type.runtime.json, opt = format.makeReadOptions(options);
    format.readMessage(type, jsonValue, opt, this);
    return this;
  }
  fromJsonString(jsonString, options) {
    let json;
    try {
      json = JSON.parse(jsonString);
    } catch (e) {
      throw new Error(`cannot decode ${this.getType().typeName} from JSON: ${e instanceof Error ? e.message : String(e)}`);
    }
    return this.fromJson(json, options);
  }
  toBinary(options) {
    const type = this.getType(), bin = type.runtime.bin, opt = bin.makeWriteOptions(options), writer = opt.writerFactory();
    bin.writeMessage(this, writer, opt);
    return writer.finish();
  }
  toJson(options) {
    const type = this.getType(), json = type.runtime.json, opt = json.makeWriteOptions(options);
    return json.writeMessage(this, opt);
  }
  toJsonString(options) {
    var _a;
    const value = this.toJson(options);
    return JSON.stringify(value, null, (_a = options === null || options === undefined ? undefined : options.prettySpaces) !== null && _a !== undefined ? _a : 0);
  }
  toJSON() {
    return this.toJson({
      emitDefaultValues: true
    });
  }
  getType() {
    return Object.getPrototypeOf(this).constructor;
  }
}

// node_modules/@bufbuild/protobuf/dist/esm/private/message-type.js
function makeMessageType(runtime, typeName, fields, opt) {
  var _a;
  const localName = (_a = opt === null || opt === undefined ? undefined : opt.localName) !== null && _a !== undefined ? _a : typeName.substring(typeName.lastIndexOf(".") + 1);
  const type = {
    [localName]: function(data) {
      runtime.util.initFields(this);
      runtime.util.initPartial(data, this);
    }
  }[localName];
  Object.setPrototypeOf(type.prototype, new Message);
  Object.assign(type, {
    runtime,
    typeName,
    fields: runtime.util.newFieldList(fields),
    fromBinary(bytes, options) {
      return new type().fromBinary(bytes, options);
    },
    fromJson(jsonValue, options) {
      return new type().fromJson(jsonValue, options);
    },
    fromJsonString(jsonString, options) {
      return new type().fromJsonString(jsonString, options);
    },
    equals(a, b) {
      return runtime.util.equals(type, a, b);
    }
  });
  return type;
}

// node_modules/@bufbuild/protobuf/dist/esm/google/varint.js
function varint64read() {
  let lowBits = 0;
  let highBits = 0;
  for (let shift = 0;shift < 28; shift += 7) {
    let b = this.buf[this.pos++];
    lowBits |= (b & 127) << shift;
    if ((b & 128) == 0) {
      this.assertBounds();
      return [lowBits, highBits];
    }
  }
  let middleByte = this.buf[this.pos++];
  lowBits |= (middleByte & 15) << 28;
  highBits = (middleByte & 112) >> 4;
  if ((middleByte & 128) == 0) {
    this.assertBounds();
    return [lowBits, highBits];
  }
  for (let shift = 3;shift <= 31; shift += 7) {
    let b = this.buf[this.pos++];
    highBits |= (b & 127) << shift;
    if ((b & 128) == 0) {
      this.assertBounds();
      return [lowBits, highBits];
    }
  }
  throw new Error("invalid varint");
}
function varint64write(lo, hi, bytes) {
  for (let i = 0;i < 28; i = i + 7) {
    const shift = lo >>> i;
    const hasNext = !(shift >>> 7 == 0 && hi == 0);
    const byte = (hasNext ? shift | 128 : shift) & 255;
    bytes.push(byte);
    if (!hasNext) {
      return;
    }
  }
  const splitBits = lo >>> 28 & 15 | (hi & 7) << 4;
  const hasMoreBits = !(hi >> 3 == 0);
  bytes.push((hasMoreBits ? splitBits | 128 : splitBits) & 255);
  if (!hasMoreBits) {
    return;
  }
  for (let i = 3;i < 31; i = i + 7) {
    const shift = hi >>> i;
    const hasNext = !(shift >>> 7 == 0);
    const byte = (hasNext ? shift | 128 : shift) & 255;
    bytes.push(byte);
    if (!hasNext) {
      return;
    }
  }
  bytes.push(hi >>> 31 & 1);
}
function int64FromString(dec) {
  const minus = dec[0] === "-";
  if (minus) {
    dec = dec.slice(1);
  }
  const base = 1e6;
  let lowBits = 0;
  let highBits = 0;
  function add1e6digit(begin, end) {
    const digit1e6 = Number(dec.slice(begin, end));
    highBits *= base;
    lowBits = lowBits * base + digit1e6;
    if (lowBits >= TWO_PWR_32_DBL) {
      highBits = highBits + (lowBits / TWO_PWR_32_DBL | 0);
      lowBits = lowBits % TWO_PWR_32_DBL;
    }
  }
  add1e6digit(-24, -18);
  add1e6digit(-18, -12);
  add1e6digit(-12, -6);
  add1e6digit(-6);
  return minus ? negate(lowBits, highBits) : newBits(lowBits, highBits);
}
function int64ToString(lo, hi) {
  let bits = newBits(lo, hi);
  const negative = bits.hi & 2147483648;
  if (negative) {
    bits = negate(bits.lo, bits.hi);
  }
  const result = uInt64ToString(bits.lo, bits.hi);
  return negative ? "-" + result : result;
}
function uInt64ToString(lo, hi) {
  ({ lo, hi } = toUnsigned(lo, hi));
  if (hi <= 2097151) {
    return String(TWO_PWR_32_DBL * hi + lo);
  }
  const low = lo & 16777215;
  const mid = (lo >>> 24 | hi << 8) & 16777215;
  const high = hi >> 16 & 65535;
  let digitA = low + mid * 6777216 + high * 6710656;
  let digitB = mid + high * 8147497;
  let digitC = high * 2;
  const base = 1e7;
  if (digitA >= base) {
    digitB += Math.floor(digitA / base);
    digitA %= base;
  }
  if (digitB >= base) {
    digitC += Math.floor(digitB / base);
    digitB %= base;
  }
  return digitC.toString() + decimalFrom1e7WithLeadingZeros(digitB) + decimalFrom1e7WithLeadingZeros(digitA);
}
var toUnsigned = function(lo, hi) {
  return { lo: lo >>> 0, hi: hi >>> 0 };
};
var newBits = function(lo, hi) {
  return { lo: lo | 0, hi: hi | 0 };
};
var negate = function(lowBits, highBits) {
  highBits = ~highBits;
  if (lowBits) {
    lowBits = ~lowBits + 1;
  } else {
    highBits += 1;
  }
  return newBits(lowBits, highBits);
};
function varint32write(value, bytes) {
  if (value >= 0) {
    while (value > 127) {
      bytes.push(value & 127 | 128);
      value = value >>> 7;
    }
    bytes.push(value);
  } else {
    for (let i = 0;i < 9; i++) {
      bytes.push(value & 127 | 128);
      value = value >> 7;
    }
    bytes.push(1);
  }
}
function varint32read() {
  let b = this.buf[this.pos++];
  let result = b & 127;
  if ((b & 128) == 0) {
    this.assertBounds();
    return result;
  }
  b = this.buf[this.pos++];
  result |= (b & 127) << 7;
  if ((b & 128) == 0) {
    this.assertBounds();
    return result;
  }
  b = this.buf[this.pos++];
  result |= (b & 127) << 14;
  if ((b & 128) == 0) {
    this.assertBounds();
    return result;
  }
  b = this.buf[this.pos++];
  result |= (b & 127) << 21;
  if ((b & 128) == 0) {
    this.assertBounds();
    return result;
  }
  b = this.buf[this.pos++];
  result |= (b & 15) << 28;
  for (let readBytes = 5;(b & 128) !== 0 && readBytes < 10; readBytes++)
    b = this.buf[this.pos++];
  if ((b & 128) != 0)
    throw new Error("invalid varint");
  this.assertBounds();
  return result >>> 0;
}
var TWO_PWR_32_DBL = 4294967296;
var decimalFrom1e7WithLeadingZeros = (digit1e7) => {
  const partial = String(digit1e7);
  return "0000000".slice(partial.length) + partial;
};

// node_modules/@bufbuild/protobuf/dist/esm/proto-int64.js
var makeInt64Support = function() {
  const dv = new DataView(new ArrayBuffer(8));
  const ok = typeof BigInt === "function" && typeof dv.getBigInt64 === "function" && typeof dv.getBigUint64 === "function" && typeof dv.setBigInt64 === "function" && typeof dv.setBigUint64 === "function" && (typeof process != "object" || typeof process.env != "object" || process.env.BUF_BIGINT_DISABLE !== "1");
  if (ok) {
    const MIN = BigInt("-9223372036854775808"), MAX = BigInt("9223372036854775807"), UMIN = BigInt("0"), UMAX = BigInt("18446744073709551615");
    return {
      zero: BigInt(0),
      supported: true,
      parse(value) {
        const bi = typeof value == "bigint" ? value : BigInt(value);
        if (bi > MAX || bi < MIN) {
          throw new Error(`int64 invalid: ${value}`);
        }
        return bi;
      },
      uParse(value) {
        const bi = typeof value == "bigint" ? value : BigInt(value);
        if (bi > UMAX || bi < UMIN) {
          throw new Error(`uint64 invalid: ${value}`);
        }
        return bi;
      },
      enc(value) {
        dv.setBigInt64(0, this.parse(value), true);
        return {
          lo: dv.getInt32(0, true),
          hi: dv.getInt32(4, true)
        };
      },
      uEnc(value) {
        dv.setBigInt64(0, this.uParse(value), true);
        return {
          lo: dv.getInt32(0, true),
          hi: dv.getInt32(4, true)
        };
      },
      dec(lo, hi) {
        dv.setInt32(0, lo, true);
        dv.setInt32(4, hi, true);
        return dv.getBigInt64(0, true);
      },
      uDec(lo, hi) {
        dv.setInt32(0, lo, true);
        dv.setInt32(4, hi, true);
        return dv.getBigUint64(0, true);
      }
    };
  }
  const assertInt64String = (value) => assert(/^-?[0-9]+$/.test(value), `int64 invalid: ${value}`);
  const assertUInt64String = (value) => assert(/^[0-9]+$/.test(value), `uint64 invalid: ${value}`);
  return {
    zero: "0",
    supported: false,
    parse(value) {
      if (typeof value != "string") {
        value = value.toString();
      }
      assertInt64String(value);
      return value;
    },
    uParse(value) {
      if (typeof value != "string") {
        value = value.toString();
      }
      assertUInt64String(value);
      return value;
    },
    enc(value) {
      if (typeof value != "string") {
        value = value.toString();
      }
      assertInt64String(value);
      return int64FromString(value);
    },
    uEnc(value) {
      if (typeof value != "string") {
        value = value.toString();
      }
      assertUInt64String(value);
      return int64FromString(value);
    },
    dec(lo, hi) {
      return int64ToString(lo, hi);
    },
    uDec(lo, hi) {
      return uInt64ToString(lo, hi);
    }
  };
};
var protoInt64 = makeInt64Support();

// node_modules/@bufbuild/protobuf/dist/esm/scalar.js
var ScalarType;
(function(ScalarType2) {
  ScalarType2[ScalarType2["DOUBLE"] = 1] = "DOUBLE";
  ScalarType2[ScalarType2["FLOAT"] = 2] = "FLOAT";
  ScalarType2[ScalarType2["INT64"] = 3] = "INT64";
  ScalarType2[ScalarType2["UINT64"] = 4] = "UINT64";
  ScalarType2[ScalarType2["INT32"] = 5] = "INT32";
  ScalarType2[ScalarType2["FIXED64"] = 6] = "FIXED64";
  ScalarType2[ScalarType2["FIXED32"] = 7] = "FIXED32";
  ScalarType2[ScalarType2["BOOL"] = 8] = "BOOL";
  ScalarType2[ScalarType2["STRING"] = 9] = "STRING";
  ScalarType2[ScalarType2["BYTES"] = 12] = "BYTES";
  ScalarType2[ScalarType2["UINT32"] = 13] = "UINT32";
  ScalarType2[ScalarType2["SFIXED32"] = 15] = "SFIXED32";
  ScalarType2[ScalarType2["SFIXED64"] = 16] = "SFIXED64";
  ScalarType2[ScalarType2["SINT32"] = 17] = "SINT32";
  ScalarType2[ScalarType2["SINT64"] = 18] = "SINT64";
})(ScalarType || (ScalarType = {}));
var LongType;
(function(LongType2) {
  LongType2[LongType2["BIGINT"] = 0] = "BIGINT";
  LongType2[LongType2["STRING"] = 1] = "STRING";
})(LongType || (LongType = {}));

// node_modules/@bufbuild/protobuf/dist/esm/private/scalars.js
function scalarEquals(type, a, b) {
  if (a === b) {
    return true;
  }
  if (type == ScalarType.BYTES) {
    if (!(a instanceof Uint8Array) || !(b instanceof Uint8Array)) {
      return false;
    }
    if (a.length !== b.length) {
      return false;
    }
    for (let i = 0;i < a.length; i++) {
      if (a[i] !== b[i]) {
        return false;
      }
    }
    return true;
  }
  switch (type) {
    case ScalarType.UINT64:
    case ScalarType.FIXED64:
    case ScalarType.INT64:
    case ScalarType.SFIXED64:
    case ScalarType.SINT64:
      return a == b;
  }
  return false;
}
function scalarZeroValue(type, longType) {
  switch (type) {
    case ScalarType.BOOL:
      return false;
    case ScalarType.UINT64:
    case ScalarType.FIXED64:
    case ScalarType.INT64:
    case ScalarType.SFIXED64:
    case ScalarType.SINT64:
      return longType == 0 ? protoInt64.zero : "0";
    case ScalarType.DOUBLE:
    case ScalarType.FLOAT:
      return 0;
    case ScalarType.BYTES:
      return new Uint8Array(0);
    case ScalarType.STRING:
      return "";
    default:
      return 0;
  }
}
function isScalarZeroValue(type, value) {
  switch (type) {
    case ScalarType.BOOL:
      return value === false;
    case ScalarType.STRING:
      return value === "";
    case ScalarType.BYTES:
      return value instanceof Uint8Array && !value.byteLength;
    default:
      return value == 0;
  }
}

// node_modules/@bufbuild/protobuf/dist/esm/binary-encoding.js
var WireType;
(function(WireType2) {
  WireType2[WireType2["Varint"] = 0] = "Varint";
  WireType2[WireType2["Bit64"] = 1] = "Bit64";
  WireType2[WireType2["LengthDelimited"] = 2] = "LengthDelimited";
  WireType2[WireType2["StartGroup"] = 3] = "StartGroup";
  WireType2[WireType2["EndGroup"] = 4] = "EndGroup";
  WireType2[WireType2["Bit32"] = 5] = "Bit32";
})(WireType || (WireType = {}));

class BinaryWriter {
  constructor(textEncoder) {
    this.stack = [];
    this.textEncoder = textEncoder !== null && textEncoder !== undefined ? textEncoder : new TextEncoder;
    this.chunks = [];
    this.buf = [];
  }
  finish() {
    this.chunks.push(new Uint8Array(this.buf));
    let len = 0;
    for (let i = 0;i < this.chunks.length; i++)
      len += this.chunks[i].length;
    let bytes = new Uint8Array(len);
    let offset = 0;
    for (let i = 0;i < this.chunks.length; i++) {
      bytes.set(this.chunks[i], offset);
      offset += this.chunks[i].length;
    }
    this.chunks = [];
    return bytes;
  }
  fork() {
    this.stack.push({ chunks: this.chunks, buf: this.buf });
    this.chunks = [];
    this.buf = [];
    return this;
  }
  join() {
    let chunk = this.finish();
    let prev = this.stack.pop();
    if (!prev)
      throw new Error("invalid state, fork stack empty");
    this.chunks = prev.chunks;
    this.buf = prev.buf;
    this.uint32(chunk.byteLength);
    return this.raw(chunk);
  }
  tag(fieldNo, type) {
    return this.uint32((fieldNo << 3 | type) >>> 0);
  }
  raw(chunk) {
    if (this.buf.length) {
      this.chunks.push(new Uint8Array(this.buf));
      this.buf = [];
    }
    this.chunks.push(chunk);
    return this;
  }
  uint32(value) {
    assertUInt32(value);
    while (value > 127) {
      this.buf.push(value & 127 | 128);
      value = value >>> 7;
    }
    this.buf.push(value);
    return this;
  }
  int32(value) {
    assertInt32(value);
    varint32write(value, this.buf);
    return this;
  }
  bool(value) {
    this.buf.push(value ? 1 : 0);
    return this;
  }
  bytes(value) {
    this.uint32(value.byteLength);
    return this.raw(value);
  }
  string(value) {
    let chunk = this.textEncoder.encode(value);
    this.uint32(chunk.byteLength);
    return this.raw(chunk);
  }
  float(value) {
    assertFloat32(value);
    let chunk = new Uint8Array(4);
    new DataView(chunk.buffer).setFloat32(0, value, true);
    return this.raw(chunk);
  }
  double(value) {
    let chunk = new Uint8Array(8);
    new DataView(chunk.buffer).setFloat64(0, value, true);
    return this.raw(chunk);
  }
  fixed32(value) {
    assertUInt32(value);
    let chunk = new Uint8Array(4);
    new DataView(chunk.buffer).setUint32(0, value, true);
    return this.raw(chunk);
  }
  sfixed32(value) {
    assertInt32(value);
    let chunk = new Uint8Array(4);
    new DataView(chunk.buffer).setInt32(0, value, true);
    return this.raw(chunk);
  }
  sint32(value) {
    assertInt32(value);
    value = (value << 1 ^ value >> 31) >>> 0;
    varint32write(value, this.buf);
    return this;
  }
  sfixed64(value) {
    let chunk = new Uint8Array(8), view = new DataView(chunk.buffer), tc = protoInt64.enc(value);
    view.setInt32(0, tc.lo, true);
    view.setInt32(4, tc.hi, true);
    return this.raw(chunk);
  }
  fixed64(value) {
    let chunk = new Uint8Array(8), view = new DataView(chunk.buffer), tc = protoInt64.uEnc(value);
    view.setInt32(0, tc.lo, true);
    view.setInt32(4, tc.hi, true);
    return this.raw(chunk);
  }
  int64(value) {
    let tc = protoInt64.enc(value);
    varint64write(tc.lo, tc.hi, this.buf);
    return this;
  }
  sint64(value) {
    let tc = protoInt64.enc(value), sign = tc.hi >> 31, lo = tc.lo << 1 ^ sign, hi = (tc.hi << 1 | tc.lo >>> 31) ^ sign;
    varint64write(lo, hi, this.buf);
    return this;
  }
  uint64(value) {
    let tc = protoInt64.uEnc(value);
    varint64write(tc.lo, tc.hi, this.buf);
    return this;
  }
}

class BinaryReader {
  constructor(buf, textDecoder) {
    this.varint64 = varint64read;
    this.uint32 = varint32read;
    this.buf = buf;
    this.len = buf.length;
    this.pos = 0;
    this.view = new DataView(buf.buffer, buf.byteOffset, buf.byteLength);
    this.textDecoder = textDecoder !== null && textDecoder !== undefined ? textDecoder : new TextDecoder;
  }
  tag() {
    let tag = this.uint32(), fieldNo = tag >>> 3, wireType = tag & 7;
    if (fieldNo <= 0 || wireType < 0 || wireType > 5)
      throw new Error("illegal tag: field no " + fieldNo + " wire type " + wireType);
    return [fieldNo, wireType];
  }
  skip(wireType) {
    let start = this.pos;
    switch (wireType) {
      case WireType.Varint:
        while (this.buf[this.pos++] & 128) {
        }
        break;
      case WireType.Bit64:
        this.pos += 4;
      case WireType.Bit32:
        this.pos += 4;
        break;
      case WireType.LengthDelimited:
        let len = this.uint32();
        this.pos += len;
        break;
      case WireType.StartGroup:
        let t;
        while ((t = this.tag()[1]) !== WireType.EndGroup) {
          this.skip(t);
        }
        break;
      default:
        throw new Error("cant skip wire type " + wireType);
    }
    this.assertBounds();
    return this.buf.subarray(start, this.pos);
  }
  assertBounds() {
    if (this.pos > this.len)
      throw new RangeError("premature EOF");
  }
  int32() {
    return this.uint32() | 0;
  }
  sint32() {
    let zze = this.uint32();
    return zze >>> 1 ^ -(zze & 1);
  }
  int64() {
    return protoInt64.dec(...this.varint64());
  }
  uint64() {
    return protoInt64.uDec(...this.varint64());
  }
  sint64() {
    let [lo, hi] = this.varint64();
    let s = -(lo & 1);
    lo = (lo >>> 1 | (hi & 1) << 31) ^ s;
    hi = hi >>> 1 ^ s;
    return protoInt64.dec(lo, hi);
  }
  bool() {
    let [lo, hi] = this.varint64();
    return lo !== 0 || hi !== 0;
  }
  fixed32() {
    return this.view.getUint32((this.pos += 4) - 4, true);
  }
  sfixed32() {
    return this.view.getInt32((this.pos += 4) - 4, true);
  }
  fixed64() {
    return protoInt64.uDec(this.sfixed32(), this.sfixed32());
  }
  sfixed64() {
    return protoInt64.dec(this.sfixed32(), this.sfixed32());
  }
  float() {
    return this.view.getFloat32((this.pos += 4) - 4, true);
  }
  double() {
    return this.view.getFloat64((this.pos += 8) - 8, true);
  }
  bytes() {
    let len = this.uint32(), start = this.pos;
    this.pos += len;
    this.assertBounds();
    return this.buf.subarray(start, start + len);
  }
  string() {
    return this.textDecoder.decode(this.bytes());
  }
}

// node_modules/@bufbuild/protobuf/dist/esm/private/extensions.js
function makeExtension(runtime, typeName, extendee, field) {
  let fi;
  return {
    typeName,
    extendee,
    get field() {
      if (!fi) {
        const i = typeof field == "function" ? field() : field;
        i.name = typeName.split(".").pop();
        i.jsonName = `[${typeName}]`;
        fi = runtime.util.newFieldList([i]).list()[0];
      }
      return fi;
    },
    runtime
  };
}
function createExtensionContainer(extension) {
  const localName = extension.field.localName;
  const container = Object.create(null);
  container[localName] = initExtensionField(extension);
  return [container, () => container[localName]];
}
var initExtensionField = function(ext) {
  const field = ext.field;
  if (field.repeated) {
    return [];
  }
  if (field.default !== undefined) {
    return field.default;
  }
  switch (field.kind) {
    case "enum":
      return field.T.values[0].no;
    case "scalar":
      return scalarZeroValue(field.T, field.L);
    case "message":
      const T = field.T, value = new T;
      return T.fieldWrapper ? T.fieldWrapper.unwrapField(value) : value;
    case "map":
      throw "map fields are not allowed to be extensions";
  }
};
function filterUnknownFields(unknownFields, field) {
  if (!field.repeated && (field.kind == "enum" || field.kind == "scalar")) {
    for (let i = unknownFields.length - 1;i >= 0; --i) {
      if (unknownFields[i].no == field.no) {
        return [unknownFields[i]];
      }
    }
    return [];
  }
  return unknownFields.filter((uf) => uf.no === field.no);
}

// node_modules/@bufbuild/protobuf/dist/esm/proto-base64.js
var encTable = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/".split("");
var decTable = [];
for (let i = 0;i < encTable.length; i++)
  decTable[encTable[i].charCodeAt(0)] = i;
decTable["-".charCodeAt(0)] = encTable.indexOf("+");
decTable["_".charCodeAt(0)] = encTable.indexOf("/");
var protoBase64 = {
  dec(base64Str) {
    let es = base64Str.length * 3 / 4;
    if (base64Str[base64Str.length - 2] == "=")
      es -= 2;
    else if (base64Str[base64Str.length - 1] == "=")
      es -= 1;
    let bytes = new Uint8Array(es), bytePos = 0, groupPos = 0, b, p = 0;
    for (let i = 0;i < base64Str.length; i++) {
      b = decTable[base64Str.charCodeAt(i)];
      if (b === undefined) {
        switch (base64Str[i]) {
          case "=":
            groupPos = 0;
          case "\n":
          case "\r":
          case "\t":
          case " ":
            continue;
          default:
            throw Error("invalid base64 string.");
        }
      }
      switch (groupPos) {
        case 0:
          p = b;
          groupPos = 1;
          break;
        case 1:
          bytes[bytePos++] = p << 2 | (b & 48) >> 4;
          p = b;
          groupPos = 2;
          break;
        case 2:
          bytes[bytePos++] = (p & 15) << 4 | (b & 60) >> 2;
          p = b;
          groupPos = 3;
          break;
        case 3:
          bytes[bytePos++] = (p & 3) << 6 | b;
          groupPos = 0;
          break;
      }
    }
    if (groupPos == 1)
      throw Error("invalid base64 string.");
    return bytes.subarray(0, bytePos);
  },
  enc(bytes) {
    let base64 = "", groupPos = 0, b, p = 0;
    for (let i = 0;i < bytes.length; i++) {
      b = bytes[i];
      switch (groupPos) {
        case 0:
          base64 += encTable[b >> 2];
          p = (b & 3) << 4;
          groupPos = 1;
          break;
        case 1:
          base64 += encTable[p | b >> 4];
          p = (b & 15) << 2;
          groupPos = 2;
          break;
        case 2:
          base64 += encTable[p | b >> 6];
          base64 += encTable[b & 63];
          groupPos = 0;
          break;
      }
    }
    if (groupPos) {
      base64 += encTable[p];
      base64 += "=";
      if (groupPos == 1)
        base64 += "=";
    }
    return base64;
  }
};

// node_modules/@bufbuild/protobuf/dist/esm/extension-accessor.js
function getExtension(message2, extension, options) {
  assertExtendee(extension, message2);
  const opt = extension.runtime.bin.makeReadOptions(options);
  const ufs = filterUnknownFields(message2.getType().runtime.bin.listUnknownFields(message2), extension.field);
  const [container, get] = createExtensionContainer(extension);
  for (const uf of ufs) {
    extension.runtime.bin.readField(container, opt.readerFactory(uf.data), extension.field, uf.wireType, opt);
  }
  return get();
}
function setExtension(message2, extension, value, options) {
  assertExtendee(extension, message2);
  const readOpt = extension.runtime.bin.makeReadOptions(options);
  const writeOpt = extension.runtime.bin.makeWriteOptions(options);
  if (hasExtension(message2, extension)) {
    const ufs = message2.getType().runtime.bin.listUnknownFields(message2).filter((uf) => uf.no != extension.field.no);
    message2.getType().runtime.bin.discardUnknownFields(message2);
    for (const uf of ufs) {
      message2.getType().runtime.bin.onUnknownField(message2, uf.no, uf.wireType, uf.data);
    }
  }
  const writer = writeOpt.writerFactory();
  let f = extension.field;
  if (!f.opt && !f.repeated && (f.kind == "enum" || f.kind == "scalar")) {
    f = Object.assign(Object.assign({}, extension.field), { opt: true });
  }
  extension.runtime.bin.writeField(f, value, writer, writeOpt);
  const reader = readOpt.readerFactory(writer.finish());
  while (reader.pos < reader.len) {
    const [no, wireType] = reader.tag();
    const data = reader.skip(wireType);
    message2.getType().runtime.bin.onUnknownField(message2, no, wireType, data);
  }
}
function hasExtension(message2, extension) {
  const messageType = message2.getType();
  return extension.extendee.typeName === messageType.typeName && !!messageType.runtime.bin.listUnknownFields(message2).find((uf) => uf.no == extension.field.no);
}
var assertExtendee = function(extension, message2) {
  assert(extension.extendee.typeName == message2.getType().typeName, `extension ${extension.typeName} can only be applied to message ${extension.extendee.typeName}`);
};

// node_modules/@bufbuild/protobuf/dist/esm/private/reflect.js
function isFieldSet(field, target) {
  const localName = field.localName;
  if (field.repeated) {
    return target[localName].length > 0;
  }
  if (field.oneof) {
    return target[field.oneof.localName].case === localName;
  }
  switch (field.kind) {
    case "enum":
    case "scalar":
      if (field.opt || field.req) {
        return target[localName] !== undefined;
      }
      if (field.kind == "enum") {
        return target[localName] !== field.T.values[0].no;
      }
      return !isScalarZeroValue(field.T, target[localName]);
    case "message":
      return target[localName] !== undefined;
    case "map":
      return Object.keys(target[localName]).length > 0;
  }
}
function clearField(field, target) {
  const localName = field.localName;
  const implicitPresence = !field.opt && !field.req;
  if (field.repeated) {
    target[localName] = [];
  } else if (field.oneof) {
    target[field.oneof.localName] = { case: undefined };
  } else {
    switch (field.kind) {
      case "map":
        target[localName] = {};
        break;
      case "enum":
        target[localName] = implicitPresence ? field.T.values[0].no : undefined;
        break;
      case "scalar":
        target[localName] = implicitPresence ? scalarZeroValue(field.T, field.L) : undefined;
        break;
      case "message":
        target[localName] = undefined;
        break;
    }
  }
}

// node_modules/@bufbuild/protobuf/dist/esm/is-message.js
function isMessage(arg, type) {
  if (arg === null || typeof arg != "object") {
    return false;
  }
  if (!Object.getOwnPropertyNames(Message.prototype).every((m) => (m in arg) && typeof arg[m] == "function")) {
    return false;
  }
  const actualType = arg.getType();
  if (actualType === null || typeof actualType != "function" || !("typeName" in actualType) || typeof actualType.typeName != "string") {
    return false;
  }
  return type === undefined ? true : actualType.typeName == type.typeName;
}

// node_modules/@bufbuild/protobuf/dist/esm/private/field-wrapper.js
function wrapField(type, value) {
  if (isMessage(value) || !type.fieldWrapper) {
    return value;
  }
  return type.fieldWrapper.wrapField(value);
}
var wktWrapperToScalarType = {
  "google.protobuf.DoubleValue": ScalarType.DOUBLE,
  "google.protobuf.FloatValue": ScalarType.FLOAT,
  "google.protobuf.Int64Value": ScalarType.INT64,
  "google.protobuf.UInt64Value": ScalarType.UINT64,
  "google.protobuf.Int32Value": ScalarType.INT32,
  "google.protobuf.UInt32Value": ScalarType.UINT32,
  "google.protobuf.BoolValue": ScalarType.BOOL,
  "google.protobuf.StringValue": ScalarType.STRING,
  "google.protobuf.BytesValue": ScalarType.BYTES
};

// node_modules/@bufbuild/protobuf/dist/esm/private/json-format.js
var makeReadOptions = function(options) {
  return options ? Object.assign(Object.assign({}, jsonReadDefaults), options) : jsonReadDefaults;
};
var makeWriteOptions = function(options) {
  return options ? Object.assign(Object.assign({}, jsonWriteDefaults), options) : jsonWriteDefaults;
};
function makeJsonFormat() {
  return {
    makeReadOptions,
    makeWriteOptions,
    readMessage(type, json, options, message3) {
      if (json == null || Array.isArray(json) || typeof json != "object") {
        throw new Error(`cannot decode message ${type.typeName} from JSON: ${debugJsonValue(json)}`);
      }
      message3 = message3 !== null && message3 !== undefined ? message3 : new type;
      const oneofSeen = new Map;
      const registry = options.typeRegistry;
      for (const [jsonKey, jsonValue] of Object.entries(json)) {
        const field = type.fields.findJsonName(jsonKey);
        if (field) {
          if (field.oneof) {
            if (jsonValue === null && field.kind == "scalar") {
              continue;
            }
            const seen = oneofSeen.get(field.oneof);
            if (seen !== undefined) {
              throw new Error(`cannot decode message ${type.typeName} from JSON: multiple keys for oneof "${field.oneof.name}" present: "${seen}", "${jsonKey}"`);
            }
            oneofSeen.set(field.oneof, jsonKey);
          }
          readField(message3, jsonValue, field, options, type);
        } else {
          let found = false;
          if ((registry === null || registry === undefined ? undefined : registry.findExtension) && jsonKey.startsWith("[") && jsonKey.endsWith("]")) {
            const ext = registry.findExtension(jsonKey.substring(1, jsonKey.length - 1));
            if (ext && ext.extendee.typeName == type.typeName) {
              found = true;
              const [container, get] = createExtensionContainer(ext);
              readField(container, jsonValue, ext.field, options, ext);
              setExtension(message3, ext, get(), options);
            }
          }
          if (!found && !options.ignoreUnknownFields) {
            throw new Error(`cannot decode message ${type.typeName} from JSON: key "${jsonKey}" is unknown`);
          }
        }
      }
      return message3;
    },
    writeMessage(message3, options) {
      const type = message3.getType();
      const json = {};
      let field;
      try {
        for (field of type.fields.byNumber()) {
          if (!isFieldSet(field, message3)) {
            if (field.req) {
              throw `required field not set`;
            }
            if (!options.emitDefaultValues) {
              continue;
            }
            if (!canEmitFieldDefaultValue(field)) {
              continue;
            }
          }
          const value = field.oneof ? message3[field.oneof.localName].value : message3[field.localName];
          const jsonValue = writeField(field, value, options);
          if (jsonValue !== undefined) {
            json[options.useProtoFieldName ? field.name : field.jsonName] = jsonValue;
          }
        }
        const registry = options.typeRegistry;
        if (registry === null || registry === undefined ? undefined : registry.findExtensionFor) {
          for (const uf of type.runtime.bin.listUnknownFields(message3)) {
            const ext = registry.findExtensionFor(type.typeName, uf.no);
            if (ext && hasExtension(message3, ext)) {
              const value = getExtension(message3, ext, options);
              const jsonValue = writeField(ext.field, value, options);
              if (jsonValue !== undefined) {
                json[ext.field.jsonName] = jsonValue;
              }
            }
          }
        }
      } catch (e) {
        const m = field ? `cannot encode field ${type.typeName}.${field.name} to JSON` : `cannot encode message ${type.typeName} to JSON`;
        const r = e instanceof Error ? e.message : String(e);
        throw new Error(m + (r.length > 0 ? `: ${r}` : ""));
      }
      return json;
    },
    readScalar(type, json, longType) {
      return readScalar(type, json, longType !== null && longType !== undefined ? longType : LongType.BIGINT, true);
    },
    writeScalar(type, value, emitDefaultValues) {
      if (value === undefined) {
        return;
      }
      if (emitDefaultValues || isScalarZeroValue(type, value)) {
        return writeScalar(type, value);
      }
      return;
    },
    debug: debugJsonValue
  };
}
var debugJsonValue = function(json) {
  if (json === null) {
    return "null";
  }
  switch (typeof json) {
    case "object":
      return Array.isArray(json) ? "array" : "object";
    case "string":
      return json.length > 100 ? "string" : `"${json.split('"').join('\\"')}"`;
    default:
      return String(json);
  }
};
var readField = function(target, jsonValue, field, options, parentType) {
  let localName = field.localName;
  if (field.repeated) {
    assert(field.kind != "map");
    if (jsonValue === null) {
      return;
    }
    if (!Array.isArray(jsonValue)) {
      throw new Error(`cannot decode field ${parentType.typeName}.${field.name} from JSON: ${debugJsonValue(jsonValue)}`);
    }
    const targetArray = target[localName];
    for (const jsonItem of jsonValue) {
      if (jsonItem === null) {
        throw new Error(`cannot decode field ${parentType.typeName}.${field.name} from JSON: ${debugJsonValue(jsonItem)}`);
      }
      switch (field.kind) {
        case "message":
          targetArray.push(field.T.fromJson(jsonItem, options));
          break;
        case "enum":
          const enumValue = readEnum(field.T, jsonItem, options.ignoreUnknownFields, true);
          if (enumValue !== tokenIgnoredUnknownEnum) {
            targetArray.push(enumValue);
          }
          break;
        case "scalar":
          try {
            targetArray.push(readScalar(field.T, jsonItem, field.L, true));
          } catch (e) {
            let m = `cannot decode field ${parentType.typeName}.${field.name} from JSON: ${debugJsonValue(jsonItem)}`;
            if (e instanceof Error && e.message.length > 0) {
              m += `: ${e.message}`;
            }
            throw new Error(m);
          }
          break;
      }
    }
  } else if (field.kind == "map") {
    if (jsonValue === null) {
      return;
    }
    if (typeof jsonValue != "object" || Array.isArray(jsonValue)) {
      throw new Error(`cannot decode field ${parentType.typeName}.${field.name} from JSON: ${debugJsonValue(jsonValue)}`);
    }
    const targetMap = target[localName];
    for (const [jsonMapKey, jsonMapValue] of Object.entries(jsonValue)) {
      if (jsonMapValue === null) {
        throw new Error(`cannot decode field ${parentType.typeName}.${field.name} from JSON: map value null`);
      }
      let key;
      try {
        key = readMapKey(field.K, jsonMapKey);
      } catch (e) {
        let m = `cannot decode map key for field ${parentType.typeName}.${field.name} from JSON: ${debugJsonValue(jsonValue)}`;
        if (e instanceof Error && e.message.length > 0) {
          m += `: ${e.message}`;
        }
        throw new Error(m);
      }
      switch (field.V.kind) {
        case "message":
          targetMap[key] = field.V.T.fromJson(jsonMapValue, options);
          break;
        case "enum":
          const enumValue = readEnum(field.V.T, jsonMapValue, options.ignoreUnknownFields, true);
          if (enumValue !== tokenIgnoredUnknownEnum) {
            targetMap[key] = enumValue;
          }
          break;
        case "scalar":
          try {
            targetMap[key] = readScalar(field.V.T, jsonMapValue, LongType.BIGINT, true);
          } catch (e) {
            let m = `cannot decode map value for field ${parentType.typeName}.${field.name} from JSON: ${debugJsonValue(jsonValue)}`;
            if (e instanceof Error && e.message.length > 0) {
              m += `: ${e.message}`;
            }
            throw new Error(m);
          }
          break;
      }
    }
  } else {
    if (field.oneof) {
      target = target[field.oneof.localName] = { case: localName };
      localName = "value";
    }
    switch (field.kind) {
      case "message":
        const messageType = field.T;
        if (jsonValue === null && messageType.typeName != "google.protobuf.Value") {
          return;
        }
        let currentValue = target[localName];
        if (isMessage(currentValue)) {
          currentValue.fromJson(jsonValue, options);
        } else {
          target[localName] = currentValue = messageType.fromJson(jsonValue, options);
          if (messageType.fieldWrapper && !field.oneof) {
            target[localName] = messageType.fieldWrapper.unwrapField(currentValue);
          }
        }
        break;
      case "enum":
        const enumValue = readEnum(field.T, jsonValue, options.ignoreUnknownFields, false);
        switch (enumValue) {
          case tokenNull:
            clearField(field, target);
            break;
          case tokenIgnoredUnknownEnum:
            break;
          default:
            target[localName] = enumValue;
            break;
        }
        break;
      case "scalar":
        try {
          const scalarValue = readScalar(field.T, jsonValue, field.L, false);
          switch (scalarValue) {
            case tokenNull:
              clearField(field, target);
              break;
            default:
              target[localName] = scalarValue;
              break;
          }
        } catch (e) {
          let m = `cannot decode field ${parentType.typeName}.${field.name} from JSON: ${debugJsonValue(jsonValue)}`;
          if (e instanceof Error && e.message.length > 0) {
            m += `: ${e.message}`;
          }
          throw new Error(m);
        }
        break;
    }
  }
};
var readMapKey = function(type, json) {
  if (type === ScalarType.BOOL) {
    switch (json) {
      case "true":
        json = true;
        break;
      case "false":
        json = false;
        break;
    }
  }
  return readScalar(type, json, LongType.BIGINT, true).toString();
};
var readScalar = function(type, json, longType, nullAsZeroValue) {
  if (json === null) {
    if (nullAsZeroValue) {
      return scalarZeroValue(type, longType);
    }
    return tokenNull;
  }
  switch (type) {
    case ScalarType.DOUBLE:
    case ScalarType.FLOAT:
      if (json === "NaN")
        return Number.NaN;
      if (json === "Infinity")
        return Number.POSITIVE_INFINITY;
      if (json === "-Infinity")
        return Number.NEGATIVE_INFINITY;
      if (json === "") {
        break;
      }
      if (typeof json == "string" && json.trim().length !== json.length) {
        break;
      }
      if (typeof json != "string" && typeof json != "number") {
        break;
      }
      const float = Number(json);
      if (Number.isNaN(float)) {
        break;
      }
      if (!Number.isFinite(float)) {
        break;
      }
      if (type == ScalarType.FLOAT)
        assertFloat32(float);
      return float;
    case ScalarType.INT32:
    case ScalarType.FIXED32:
    case ScalarType.SFIXED32:
    case ScalarType.SINT32:
    case ScalarType.UINT32:
      let int32;
      if (typeof json == "number")
        int32 = json;
      else if (typeof json == "string" && json.length > 0) {
        if (json.trim().length === json.length)
          int32 = Number(json);
      }
      if (int32 === undefined)
        break;
      if (type == ScalarType.UINT32)
        assertUInt32(int32);
      else
        assertInt32(int32);
      return int32;
    case ScalarType.INT64:
    case ScalarType.SFIXED64:
    case ScalarType.SINT64:
      if (typeof json != "number" && typeof json != "string")
        break;
      const long = protoInt64.parse(json);
      return longType ? long.toString() : long;
    case ScalarType.FIXED64:
    case ScalarType.UINT64:
      if (typeof json != "number" && typeof json != "string")
        break;
      const uLong = protoInt64.uParse(json);
      return longType ? uLong.toString() : uLong;
    case ScalarType.BOOL:
      if (typeof json !== "boolean")
        break;
      return json;
    case ScalarType.STRING:
      if (typeof json !== "string") {
        break;
      }
      try {
        encodeURIComponent(json);
      } catch (e) {
        throw new Error("invalid UTF8");
      }
      return json;
    case ScalarType.BYTES:
      if (json === "")
        return new Uint8Array(0);
      if (typeof json !== "string")
        break;
      return protoBase64.dec(json);
  }
  throw new Error;
};
var readEnum = function(type, json, ignoreUnknownFields, nullAsZeroValue) {
  if (json === null) {
    if (type.typeName == "google.protobuf.NullValue") {
      return 0;
    }
    return nullAsZeroValue ? type.values[0].no : tokenNull;
  }
  switch (typeof json) {
    case "number":
      if (Number.isInteger(json)) {
        return json;
      }
      break;
    case "string":
      const value = type.findName(json);
      if (value !== undefined) {
        return value.no;
      }
      if (ignoreUnknownFields) {
        return tokenIgnoredUnknownEnum;
      }
      break;
  }
  throw new Error(`cannot decode enum ${type.typeName} from JSON: ${debugJsonValue(json)}`);
};
var canEmitFieldDefaultValue = function(field) {
  if (field.repeated || field.kind == "map") {
    return true;
  }
  if (field.oneof) {
    return false;
  }
  if (field.kind == "message") {
    return false;
  }
  if (field.opt || field.req) {
    return false;
  }
  return true;
};
var writeField = function(field, value, options) {
  if (field.kind == "map") {
    assert(typeof value == "object" && value != null);
    const jsonObj = {};
    const entries = Object.entries(value);
    switch (field.V.kind) {
      case "scalar":
        for (const [entryKey, entryValue] of entries) {
          jsonObj[entryKey.toString()] = writeScalar(field.V.T, entryValue);
        }
        break;
      case "message":
        for (const [entryKey, entryValue] of entries) {
          jsonObj[entryKey.toString()] = entryValue.toJson(options);
        }
        break;
      case "enum":
        const enumType = field.V.T;
        for (const [entryKey, entryValue] of entries) {
          jsonObj[entryKey.toString()] = writeEnum(enumType, entryValue, options.enumAsInteger);
        }
        break;
    }
    return options.emitDefaultValues || entries.length > 0 ? jsonObj : undefined;
  }
  if (field.repeated) {
    assert(Array.isArray(value));
    const jsonArr = [];
    switch (field.kind) {
      case "scalar":
        for (let i = 0;i < value.length; i++) {
          jsonArr.push(writeScalar(field.T, value[i]));
        }
        break;
      case "enum":
        for (let i = 0;i < value.length; i++) {
          jsonArr.push(writeEnum(field.T, value[i], options.enumAsInteger));
        }
        break;
      case "message":
        for (let i = 0;i < value.length; i++) {
          jsonArr.push(value[i].toJson(options));
        }
        break;
    }
    return options.emitDefaultValues || jsonArr.length > 0 ? jsonArr : undefined;
  }
  switch (field.kind) {
    case "scalar":
      return writeScalar(field.T, value);
    case "enum":
      return writeEnum(field.T, value, options.enumAsInteger);
    case "message":
      return wrapField(field.T, value).toJson(options);
  }
};
var writeEnum = function(type, value, enumAsInteger) {
  var _a;
  assert(typeof value == "number");
  if (type.typeName == "google.protobuf.NullValue") {
    return null;
  }
  if (enumAsInteger) {
    return value;
  }
  const val = type.findNumber(value);
  return (_a = val === null || val === undefined ? undefined : val.name) !== null && _a !== undefined ? _a : value;
};
var writeScalar = function(type, value) {
  switch (type) {
    case ScalarType.INT32:
    case ScalarType.SFIXED32:
    case ScalarType.SINT32:
    case ScalarType.FIXED32:
    case ScalarType.UINT32:
      assert(typeof value == "number");
      return value;
    case ScalarType.FLOAT:
    case ScalarType.DOUBLE:
      assert(typeof value == "number");
      if (Number.isNaN(value))
        return "NaN";
      if (value === Number.POSITIVE_INFINITY)
        return "Infinity";
      if (value === Number.NEGATIVE_INFINITY)
        return "-Infinity";
      return value;
    case ScalarType.STRING:
      assert(typeof value == "string");
      return value;
    case ScalarType.BOOL:
      assert(typeof value == "boolean");
      return value;
    case ScalarType.UINT64:
    case ScalarType.FIXED64:
    case ScalarType.INT64:
    case ScalarType.SFIXED64:
    case ScalarType.SINT64:
      assert(typeof value == "bigint" || typeof value == "string" || typeof value == "number");
      return value.toString();
    case ScalarType.BYTES:
      assert(value instanceof Uint8Array);
      return protoBase64.enc(value);
  }
};
var jsonReadDefaults = {
  ignoreUnknownFields: false
};
var jsonWriteDefaults = {
  emitDefaultValues: false,
  enumAsInteger: false,
  useProtoFieldName: false,
  prettySpaces: 0
};
var tokenNull = Symbol();
var tokenIgnoredUnknownEnum = Symbol();

// node_modules/@bufbuild/protobuf/dist/esm/private/binary-format.js
var makeReadOptions2 = function(options) {
  return options ? Object.assign(Object.assign({}, readDefaults), options) : readDefaults;
};
var makeWriteOptions2 = function(options) {
  return options ? Object.assign(Object.assign({}, writeDefaults), options) : writeDefaults;
};
function makeBinaryFormat() {
  return {
    makeReadOptions: makeReadOptions2,
    makeWriteOptions: makeWriteOptions2,
    listUnknownFields(message3) {
      var _a;
      return (_a = message3[unknownFieldsSymbol]) !== null && _a !== undefined ? _a : [];
    },
    discardUnknownFields(message3) {
      delete message3[unknownFieldsSymbol];
    },
    writeUnknownFields(message3, writer) {
      const m = message3;
      const c = m[unknownFieldsSymbol];
      if (c) {
        for (const f of c) {
          writer.tag(f.no, f.wireType).raw(f.data);
        }
      }
    },
    onUnknownField(message3, no, wireType, data) {
      const m = message3;
      if (!Array.isArray(m[unknownFieldsSymbol])) {
        m[unknownFieldsSymbol] = [];
      }
      m[unknownFieldsSymbol].push({ no, wireType, data });
    },
    readMessage(message3, reader, lengthOrEndTagFieldNo, options, delimitedMessageEncoding) {
      const type = message3.getType();
      const end = delimitedMessageEncoding ? reader.len : reader.pos + lengthOrEndTagFieldNo;
      let fieldNo, wireType;
      while (reader.pos < end) {
        [fieldNo, wireType] = reader.tag();
        if (wireType == WireType.EndGroup) {
          break;
        }
        const field = type.fields.find(fieldNo);
        if (!field) {
          const data = reader.skip(wireType);
          if (options.readUnknownFields) {
            this.onUnknownField(message3, fieldNo, wireType, data);
          }
          continue;
        }
        readField2(message3, reader, field, wireType, options);
      }
      if (delimitedMessageEncoding && (wireType != WireType.EndGroup || fieldNo !== lengthOrEndTagFieldNo)) {
        throw new Error(`invalid end group tag`);
      }
    },
    readField: readField2,
    writeMessage(message3, writer, options) {
      const type = message3.getType();
      for (const field of type.fields.byNumber()) {
        if (!isFieldSet(field, message3)) {
          if (field.req) {
            throw new Error(`cannot encode field ${type.typeName}.${field.name} to binary: required field not set`);
          }
          continue;
        }
        const value = field.oneof ? message3[field.oneof.localName].value : message3[field.localName];
        writeField2(field, value, writer, options);
      }
      if (options.writeUnknownFields) {
        this.writeUnknownFields(message3, writer);
      }
      return writer;
    },
    writeField(field, value, writer, options) {
      if (value === undefined) {
        return;
      }
      writeField2(field, value, writer, options);
    }
  };
}
var readField2 = function(target, reader, field, wireType, options) {
  let { repeated, localName } = field;
  if (field.oneof) {
    target = target[field.oneof.localName];
    if (target.case != localName) {
      delete target.value;
    }
    target.case = localName;
    localName = "value";
  }
  switch (field.kind) {
    case "scalar":
    case "enum":
      const scalarType = field.kind == "enum" ? ScalarType.INT32 : field.T;
      let read = readScalar2;
      if (field.kind == "scalar" && field.L > 0) {
        read = readScalarLTString;
      }
      if (repeated) {
        let arr = target[localName];
        const isPacked = wireType == WireType.LengthDelimited && scalarType != ScalarType.STRING && scalarType != ScalarType.BYTES;
        if (isPacked) {
          let e = reader.uint32() + reader.pos;
          while (reader.pos < e) {
            arr.push(read(reader, scalarType));
          }
        } else {
          arr.push(read(reader, scalarType));
        }
      } else {
        target[localName] = read(reader, scalarType);
      }
      break;
    case "message":
      const messageType = field.T;
      if (repeated) {
        target[localName].push(readMessageField(reader, new messageType, options, field));
      } else {
        if (isMessage(target[localName])) {
          readMessageField(reader, target[localName], options, field);
        } else {
          target[localName] = readMessageField(reader, new messageType, options, field);
          if (messageType.fieldWrapper && !field.oneof && !field.repeated) {
            target[localName] = messageType.fieldWrapper.unwrapField(target[localName]);
          }
        }
      }
      break;
    case "map":
      let [mapKey, mapVal] = readMapEntry(field, reader, options);
      target[localName][mapKey] = mapVal;
      break;
  }
};
var readMessageField = function(reader, message3, options, field) {
  const format = message3.getType().runtime.bin;
  const delimited = field === null || field === undefined ? undefined : field.delimited;
  format.readMessage(message3, reader, delimited ? field.no : reader.uint32(), options, delimited);
  return message3;
};
var readMapEntry = function(field, reader, options) {
  const length = reader.uint32(), end = reader.pos + length;
  let key, val;
  while (reader.pos < end) {
    const [fieldNo] = reader.tag();
    switch (fieldNo) {
      case 1:
        key = readScalar2(reader, field.K);
        break;
      case 2:
        switch (field.V.kind) {
          case "scalar":
            val = readScalar2(reader, field.V.T);
            break;
          case "enum":
            val = reader.int32();
            break;
          case "message":
            val = readMessageField(reader, new field.V.T, options, undefined);
            break;
        }
        break;
    }
  }
  if (key === undefined) {
    key = scalarZeroValue(field.K, LongType.BIGINT);
  }
  if (typeof key != "string" && typeof key != "number") {
    key = key.toString();
  }
  if (val === undefined) {
    switch (field.V.kind) {
      case "scalar":
        val = scalarZeroValue(field.V.T, LongType.BIGINT);
        break;
      case "enum":
        val = field.V.T.values[0].no;
        break;
      case "message":
        val = new field.V.T;
        break;
    }
  }
  return [key, val];
};
var readScalarLTString = function(reader, type) {
  const v = readScalar2(reader, type);
  return typeof v == "bigint" ? v.toString() : v;
};
var readScalar2 = function(reader, type) {
  switch (type) {
    case ScalarType.STRING:
      return reader.string();
    case ScalarType.BOOL:
      return reader.bool();
    case ScalarType.DOUBLE:
      return reader.double();
    case ScalarType.FLOAT:
      return reader.float();
    case ScalarType.INT32:
      return reader.int32();
    case ScalarType.INT64:
      return reader.int64();
    case ScalarType.UINT64:
      return reader.uint64();
    case ScalarType.FIXED64:
      return reader.fixed64();
    case ScalarType.BYTES:
      return reader.bytes();
    case ScalarType.FIXED32:
      return reader.fixed32();
    case ScalarType.SFIXED32:
      return reader.sfixed32();
    case ScalarType.SFIXED64:
      return reader.sfixed64();
    case ScalarType.SINT64:
      return reader.sint64();
    case ScalarType.UINT32:
      return reader.uint32();
    case ScalarType.SINT32:
      return reader.sint32();
  }
};
var writeField2 = function(field, value, writer, options) {
  assert(value !== undefined);
  const repeated = field.repeated;
  switch (field.kind) {
    case "scalar":
    case "enum":
      let scalarType = field.kind == "enum" ? ScalarType.INT32 : field.T;
      if (repeated) {
        assert(Array.isArray(value));
        if (field.packed) {
          writePacked(writer, scalarType, field.no, value);
        } else {
          for (const item of value) {
            writeScalar2(writer, scalarType, field.no, item);
          }
        }
      } else {
        writeScalar2(writer, scalarType, field.no, value);
      }
      break;
    case "message":
      if (repeated) {
        assert(Array.isArray(value));
        for (const item of value) {
          writeMessageField(writer, options, field, item);
        }
      } else {
        writeMessageField(writer, options, field, value);
      }
      break;
    case "map":
      assert(typeof value == "object" && value != null);
      for (const [key, val] of Object.entries(value)) {
        writeMapEntry(writer, options, field, key, val);
      }
      break;
  }
};
function writeMapEntry(writer, options, field, key, value) {
  writer.tag(field.no, WireType.LengthDelimited);
  writer.fork();
  let keyValue = key;
  switch (field.K) {
    case ScalarType.INT32:
    case ScalarType.FIXED32:
    case ScalarType.UINT32:
    case ScalarType.SFIXED32:
    case ScalarType.SINT32:
      keyValue = Number.parseInt(key);
      break;
    case ScalarType.BOOL:
      assert(key == "true" || key == "false");
      keyValue = key == "true";
      break;
  }
  writeScalar2(writer, field.K, 1, keyValue);
  switch (field.V.kind) {
    case "scalar":
      writeScalar2(writer, field.V.T, 2, value);
      break;
    case "enum":
      writeScalar2(writer, ScalarType.INT32, 2, value);
      break;
    case "message":
      assert(value !== undefined);
      writer.tag(2, WireType.LengthDelimited).bytes(value.toBinary(options));
      break;
  }
  writer.join();
}
var writeMessageField = function(writer, options, field, value) {
  const message3 = wrapField(field.T, value);
  if (field.delimited)
    writer.tag(field.no, WireType.StartGroup).raw(message3.toBinary(options)).tag(field.no, WireType.EndGroup);
  else
    writer.tag(field.no, WireType.LengthDelimited).bytes(message3.toBinary(options));
};
var writeScalar2 = function(writer, type, fieldNo, value) {
  assert(value !== undefined);
  let [wireType, method] = scalarTypeInfo(type);
  writer.tag(fieldNo, wireType)[method](value);
};
var writePacked = function(writer, type, fieldNo, value) {
  if (!value.length) {
    return;
  }
  writer.tag(fieldNo, WireType.LengthDelimited).fork();
  let [, method] = scalarTypeInfo(type);
  for (let i = 0;i < value.length; i++) {
    writer[method](value[i]);
  }
  writer.join();
};
var scalarTypeInfo = function(type) {
  let wireType = WireType.Varint;
  switch (type) {
    case ScalarType.BYTES:
    case ScalarType.STRING:
      wireType = WireType.LengthDelimited;
      break;
    case ScalarType.DOUBLE:
    case ScalarType.FIXED64:
    case ScalarType.SFIXED64:
      wireType = WireType.Bit64;
      break;
    case ScalarType.FIXED32:
    case ScalarType.SFIXED32:
    case ScalarType.FLOAT:
      wireType = WireType.Bit32;
      break;
  }
  const method = ScalarType[type].toLowerCase();
  return [wireType, method];
};
var unknownFieldsSymbol = Symbol("@bufbuild/protobuf/unknown-fields");
var readDefaults = {
  readUnknownFields: true,
  readerFactory: (bytes) => new BinaryReader(bytes)
};
var writeDefaults = {
  writeUnknownFields: true,
  writerFactory: () => new BinaryWriter
};

// node_modules/@bufbuild/protobuf/dist/esm/private/util-common.js
function makeUtilCommon() {
  return {
    setEnumType,
    initPartial(source, target) {
      if (source === undefined) {
        return;
      }
      const type = target.getType();
      for (const member of type.fields.byMember()) {
        const localName = member.localName, t = target, s = source;
        if (s[localName] === undefined) {
          continue;
        }
        switch (member.kind) {
          case "oneof":
            const sk = s[localName].case;
            if (sk === undefined) {
              continue;
            }
            const sourceField = member.findField(sk);
            let val = s[localName].value;
            if (sourceField && sourceField.kind == "message" && !isMessage(val, sourceField.T)) {
              val = new sourceField.T(val);
            } else if (sourceField && sourceField.kind === "scalar" && sourceField.T === ScalarType.BYTES) {
              val = toU8Arr(val);
            }
            t[localName] = { case: sk, value: val };
            break;
          case "scalar":
          case "enum":
            let copy = s[localName];
            if (member.T === ScalarType.BYTES) {
              copy = member.repeated ? copy.map(toU8Arr) : toU8Arr(copy);
            }
            t[localName] = copy;
            break;
          case "map":
            switch (member.V.kind) {
              case "scalar":
              case "enum":
                if (member.V.T === ScalarType.BYTES) {
                  for (const [k, v] of Object.entries(s[localName])) {
                    t[localName][k] = toU8Arr(v);
                  }
                } else {
                  Object.assign(t[localName], s[localName]);
                }
                break;
              case "message":
                const messageType = member.V.T;
                for (const k of Object.keys(s[localName])) {
                  let val2 = s[localName][k];
                  if (!messageType.fieldWrapper) {
                    val2 = new messageType(val2);
                  }
                  t[localName][k] = val2;
                }
                break;
            }
            break;
          case "message":
            const mt = member.T;
            if (member.repeated) {
              t[localName] = s[localName].map((val2) => isMessage(val2, mt) ? val2 : new mt(val2));
            } else {
              const val2 = s[localName];
              if (mt.fieldWrapper) {
                if (mt.typeName === "google.protobuf.BytesValue") {
                  t[localName] = toU8Arr(val2);
                } else {
                  t[localName] = val2;
                }
              } else {
                t[localName] = isMessage(val2, mt) ? val2 : new mt(val2);
              }
            }
            break;
        }
      }
    },
    equals(type, a, b) {
      if (a === b) {
        return true;
      }
      if (!a || !b) {
        return false;
      }
      return type.fields.byMember().every((m) => {
        const va = a[m.localName];
        const vb = b[m.localName];
        if (m.repeated) {
          if (va.length !== vb.length) {
            return false;
          }
          switch (m.kind) {
            case "message":
              return va.every((a2, i) => m.T.equals(a2, vb[i]));
            case "scalar":
              return va.every((a2, i) => scalarEquals(m.T, a2, vb[i]));
            case "enum":
              return va.every((a2, i) => scalarEquals(ScalarType.INT32, a2, vb[i]));
          }
          throw new Error(`repeated cannot contain ${m.kind}`);
        }
        switch (m.kind) {
          case "message":
            return m.T.equals(va, vb);
          case "enum":
            return scalarEquals(ScalarType.INT32, va, vb);
          case "scalar":
            return scalarEquals(m.T, va, vb);
          case "oneof":
            if (va.case !== vb.case) {
              return false;
            }
            const s = m.findField(va.case);
            if (s === undefined) {
              return true;
            }
            switch (s.kind) {
              case "message":
                return s.T.equals(va.value, vb.value);
              case "enum":
                return scalarEquals(ScalarType.INT32, va.value, vb.value);
              case "scalar":
                return scalarEquals(s.T, va.value, vb.value);
            }
            throw new Error(`oneof cannot contain ${s.kind}`);
          case "map":
            const keys = Object.keys(va).concat(Object.keys(vb));
            switch (m.V.kind) {
              case "message":
                const messageType = m.V.T;
                return keys.every((k) => messageType.equals(va[k], vb[k]));
              case "enum":
                return keys.every((k) => scalarEquals(ScalarType.INT32, va[k], vb[k]));
              case "scalar":
                const scalarType = m.V.T;
                return keys.every((k) => scalarEquals(scalarType, va[k], vb[k]));
            }
            break;
        }
      });
    },
    clone(message3) {
      const type = message3.getType(), target = new type, any = target;
      for (const member of type.fields.byMember()) {
        const source = message3[member.localName];
        let copy;
        if (member.repeated) {
          copy = source.map(cloneSingularField);
        } else if (member.kind == "map") {
          copy = any[member.localName];
          for (const [key, v] of Object.entries(source)) {
            copy[key] = cloneSingularField(v);
          }
        } else if (member.kind == "oneof") {
          const f = member.findField(source.case);
          copy = f ? { case: source.case, value: cloneSingularField(source.value) } : { case: undefined };
        } else {
          copy = cloneSingularField(source);
        }
        any[member.localName] = copy;
      }
      return target;
    }
  };
}
var cloneSingularField = function(value) {
  if (value === undefined) {
    return value;
  }
  if (isMessage(value)) {
    return value.clone();
  }
  if (value instanceof Uint8Array) {
    const c = new Uint8Array(value.byteLength);
    c.set(value);
    return c;
  }
  return value;
};
var toU8Arr = function(input) {
  return input instanceof Uint8Array ? input : new Uint8Array(input);
};

// node_modules/@bufbuild/protobuf/dist/esm/private/proto-runtime.js
function makeProtoRuntime(syntax, newFieldList, initFields) {
  return {
    syntax,
    json: makeJsonFormat(),
    bin: makeBinaryFormat(),
    util: Object.assign(Object.assign({}, makeUtilCommon()), {
      newFieldList,
      initFields
    }),
    makeMessageType(typeName, fields, opt) {
      return makeMessageType(this, typeName, fields, opt);
    },
    makeEnum,
    makeEnumType,
    getEnumType,
    makeExtension(typeName, extendee, field) {
      return makeExtension(this, typeName, extendee, field);
    }
  };
}

// node_modules/@bufbuild/protobuf/dist/esm/private/field-list.js
class InternalFieldList {
  constructor(fields, normalizer) {
    this._fields = fields;
    this._normalizer = normalizer;
  }
  findJsonName(jsonName) {
    if (!this.jsonNames) {
      const t = {};
      for (const f of this.list()) {
        t[f.jsonName] = t[f.name] = f;
      }
      this.jsonNames = t;
    }
    return this.jsonNames[jsonName];
  }
  find(fieldNo) {
    if (!this.numbers) {
      const t = {};
      for (const f of this.list()) {
        t[f.no] = f;
      }
      this.numbers = t;
    }
    return this.numbers[fieldNo];
  }
  list() {
    if (!this.all) {
      this.all = this._normalizer(this._fields);
    }
    return this.all;
  }
  byNumber() {
    if (!this.numbersAsc) {
      this.numbersAsc = this.list().concat().sort((a, b) => a.no - b.no);
    }
    return this.numbersAsc;
  }
  byMember() {
    if (!this.members) {
      this.members = [];
      const a = this.members;
      let o;
      for (const f of this.list()) {
        if (f.oneof) {
          if (f.oneof !== o) {
            o = f.oneof;
            a.push(o);
          }
        } else {
          a.push(f);
        }
      }
    }
    return this.members;
  }
}

// node_modules/@bufbuild/protobuf/dist/esm/private/names.js
function localFieldName(protoName, inOneof) {
  const name = protoCamelCase(protoName);
  if (inOneof) {
    return name;
  }
  return safeObjectProperty(safeMessageProperty(name));
}
function localOneofName(protoName) {
  return localFieldName(protoName, false);
}
var protoCamelCase = function(snakeCase) {
  let capNext = false;
  const b = [];
  for (let i = 0;i < snakeCase.length; i++) {
    let c = snakeCase.charAt(i);
    switch (c) {
      case "_":
        capNext = true;
        break;
      case "0":
      case "1":
      case "2":
      case "3":
      case "4":
      case "5":
      case "6":
      case "7":
      case "8":
      case "9":
        b.push(c);
        capNext = false;
        break;
      default:
        if (capNext) {
          capNext = false;
          c = c.toUpperCase();
        }
        b.push(c);
        break;
    }
  }
  return b.join("");
};
var fieldJsonName = protoCamelCase;
var reservedIdentifiers = new Set([
  "break",
  "case",
  "catch",
  "class",
  "const",
  "continue",
  "debugger",
  "default",
  "delete",
  "do",
  "else",
  "export",
  "extends",
  "false",
  "finally",
  "for",
  "function",
  "if",
  "import",
  "in",
  "instanceof",
  "new",
  "null",
  "return",
  "super",
  "switch",
  "this",
  "throw",
  "true",
  "try",
  "typeof",
  "var",
  "void",
  "while",
  "with",
  "yield",
  "enum",
  "implements",
  "interface",
  "let",
  "package",
  "private",
  "protected",
  "public",
  "static",
  "Object",
  "bigint",
  "number",
  "boolean",
  "string",
  "object",
  "globalThis",
  "Uint8Array",
  "Partial"
]);
var reservedObjectProperties = new Set([
  "constructor",
  "toString",
  "toJSON",
  "valueOf"
]);
var reservedMessageProperties = new Set([
  "getType",
  "clone",
  "equals",
  "fromBinary",
  "fromJson",
  "fromJsonString",
  "toBinary",
  "toJson",
  "toJsonString",
  "toObject"
]);
var fallback = (name) => `${name}\$`;
var safeMessageProperty = (name) => {
  if (reservedMessageProperties.has(name)) {
    return fallback(name);
  }
  return name;
};
var safeObjectProperty = (name) => {
  if (reservedObjectProperties.has(name)) {
    return fallback(name);
  }
  return name;
};

// node_modules/@bufbuild/protobuf/dist/esm/private/field.js
class InternalOneofInfo {
  constructor(name) {
    this.kind = "oneof";
    this.repeated = false;
    this.packed = false;
    this.opt = false;
    this.req = false;
    this.default = undefined;
    this.fields = [];
    this.name = name;
    this.localName = localOneofName(name);
  }
  addField(field) {
    assert(field.oneof === this, `field ${field.name} not one of ${this.name}`);
    this.fields.push(field);
  }
  findField(localName) {
    if (!this._lookup) {
      this._lookup = Object.create(null);
      for (let i = 0;i < this.fields.length; i++) {
        this._lookup[this.fields[i].localName] = this.fields[i];
      }
    }
    return this._lookup[localName];
  }
}

// node_modules/@bufbuild/protobuf/dist/esm/private/field-normalize.js
function normalizeFieldInfos(fieldInfos, packedByDefault) {
  var _a, _b, _c, _d, _e, _f;
  const r = [];
  let o;
  for (const field2 of typeof fieldInfos == "function" ? fieldInfos() : fieldInfos) {
    const f = field2;
    f.localName = localFieldName(field2.name, field2.oneof !== undefined);
    f.jsonName = (_a = field2.jsonName) !== null && _a !== undefined ? _a : fieldJsonName(field2.name);
    f.repeated = (_b = field2.repeated) !== null && _b !== undefined ? _b : false;
    if (field2.kind == "scalar") {
      f.L = (_c = field2.L) !== null && _c !== undefined ? _c : LongType.BIGINT;
    }
    f.delimited = (_d = field2.delimited) !== null && _d !== undefined ? _d : false;
    f.req = (_e = field2.req) !== null && _e !== undefined ? _e : false;
    f.opt = (_f = field2.opt) !== null && _f !== undefined ? _f : false;
    if (field2.packed === undefined) {
      if (packedByDefault) {
        f.packed = field2.kind == "enum" || field2.kind == "scalar" && field2.T != ScalarType.BYTES && field2.T != ScalarType.STRING;
      } else {
        f.packed = false;
      }
    }
    if (field2.oneof !== undefined) {
      const ooname = typeof field2.oneof == "string" ? field2.oneof : field2.oneof.name;
      if (!o || o.name != ooname) {
        o = new InternalOneofInfo(ooname);
      }
      f.oneof = o;
      o.addField(f);
    }
    r.push(f);
  }
  return r;
}

// node_modules/@bufbuild/protobuf/dist/esm/proto3.js
var proto3 = makeProtoRuntime("proto3", (fields) => {
  return new InternalFieldList(fields, (source) => normalizeFieldInfos(source, true));
}, (target) => {
  for (const member of target.getType().fields.byMember()) {
    if (member.opt) {
      continue;
    }
    const name = member.localName, t = target;
    if (member.repeated) {
      t[name] = [];
      continue;
    }
    switch (member.kind) {
      case "oneof":
        t[name] = { case: undefined };
        break;
      case "enum":
        t[name] = 0;
        break;
      case "map":
        t[name] = {};
        break;
      case "scalar":
        t[name] = scalarZeroValue(member.T, member.L);
        break;
      case "message":
        break;
    }
  }
});
// node_modules/@bufbuild/protobuf/dist/esm/service-type.js
var MethodKind;
(function(MethodKind2) {
  MethodKind2[MethodKind2["Unary"] = 0] = "Unary";
  MethodKind2[MethodKind2["ServerStreaming"] = 1] = "ServerStreaming";
  MethodKind2[MethodKind2["ClientStreaming"] = 2] = "ClientStreaming";
  MethodKind2[MethodKind2["BiDiStreaming"] = 3] = "BiDiStreaming";
})(MethodKind || (MethodKind = {}));
var MethodIdempotency;
(function(MethodIdempotency2) {
  MethodIdempotency2[MethodIdempotency2["NoSideEffects"] = 1] = "NoSideEffects";
  MethodIdempotency2[MethodIdempotency2["Idempotent"] = 2] = "Idempotent";
})(MethodIdempotency || (MethodIdempotency = {}));
// node_modules/@connectrpc/connect/dist/esm/http-headers.js
function appendHeaders(...headers) {
  const h = new Headers;
  for (const e of headers) {
    e.forEach((value, key) => {
      h.append(key, value);
    });
  }
  return h;
}
// node_modules/@connectrpc/connect/dist/esm/any-client.js
function makeAnyClient(service, createMethod) {
  const client = {};
  for (const [localName, methodInfo] of Object.entries(service.methods)) {
    const method = createMethod(Object.assign(Object.assign({}, methodInfo), {
      localName,
      service
    }));
    if (method != null) {
      client[localName] = method;
    }
  }
  return client;
}

// node_modules/@connectrpc/connect/dist/esm/protocol/envelope.js
function createEnvelopeReadableStream(stream) {
  let reader;
  let buffer = new Uint8Array(0);
  function append(chunk) {
    const n = new Uint8Array(buffer.length + chunk.length);
    n.set(buffer);
    n.set(chunk, buffer.length);
    buffer = n;
  }
  return new ReadableStream({
    start() {
      reader = stream.getReader();
    },
    async pull(controller) {
      let header = undefined;
      for (;; ) {
        if (header === undefined && buffer.byteLength >= 5) {
          let length = 0;
          for (let i = 1;i < 5; i++) {
            length = (length << 8) + buffer[i];
          }
          header = { flags: buffer[0], length };
        }
        if (header !== undefined && buffer.byteLength >= header.length + 5) {
          break;
        }
        const result = await reader.read();
        if (result.done) {
          break;
        }
        append(result.value);
      }
      if (header === undefined) {
        if (buffer.byteLength == 0) {
          controller.close();
          return;
        }
        controller.error(new ConnectError("premature end of stream", Code.DataLoss));
        return;
      }
      const data = buffer.subarray(5, 5 + header.length);
      buffer = buffer.subarray(5 + header.length);
      controller.enqueue({
        flags: header.flags,
        data
      });
    }
  });
}
function encodeEnvelope(flags, data) {
  const bytes = new Uint8Array(data.length + 5);
  bytes.set(data, 5);
  const v = new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength);
  v.setUint8(0, flags);
  v.setUint32(1, data.length);
  return bytes;
}

// node_modules/@connectrpc/connect/dist/esm/protocol/async-iterable.js
function createAsyncIterable(items) {
  return __asyncGenerator(this, arguments, function* createAsyncIterable_1() {
    yield __await(yield* __asyncDelegator(__asyncValues(items)));
  });
}
var __asyncValues = function(o) {
  if (!Symbol.asyncIterator)
    throw new TypeError("Symbol.asyncIterator is not defined.");
  var m = o[Symbol.asyncIterator], i;
  return m ? m.call(o) : (o = typeof __values === "function" ? __values(o) : o[Symbol.iterator](), i = {}, verb("next"), verb("throw"), verb("return"), i[Symbol.asyncIterator] = function() {
    return this;
  }, i);
  function verb(n) {
    i[n] = o[n] && function(v) {
      return new Promise(function(resolve, reject) {
        v = o[n](v), settle(resolve, reject, v.done, v.value);
      });
    };
  }
  function settle(resolve, reject, d, v) {
    Promise.resolve(v).then(function(v2) {
      resolve({ value: v2, done: d });
    }, reject);
  }
};
var __await = function(v) {
  return this instanceof __await ? (this.v = v, this) : new __await(v);
};
var __asyncGenerator = function(thisArg, _arguments, generator) {
  if (!Symbol.asyncIterator)
    throw new TypeError("Symbol.asyncIterator is not defined.");
  var g = generator.apply(thisArg, _arguments || []), i, q = [];
  return i = {}, verb("next"), verb("throw"), verb("return", awaitReturn), i[Symbol.asyncIterator] = function() {
    return this;
  }, i;
  function awaitReturn(f) {
    return function(v) {
      return Promise.resolve(v).then(f, reject);
    };
  }
  function verb(n, f) {
    if (g[n]) {
      i[n] = function(v) {
        return new Promise(function(a, b) {
          q.push([n, v, a, b]) > 1 || resume(n, v);
        });
      };
      if (f)
        i[n] = f(i[n]);
    }
  }
  function resume(n, v) {
    try {
      step(g[n](v));
    } catch (e) {
      settle(q[0][3], e);
    }
  }
  function step(r) {
    r.value instanceof __await ? Promise.resolve(r.value.v).then(fulfill, reject) : settle(q[0][2], r);
  }
  function fulfill(value) {
    resume("next", value);
  }
  function reject(value) {
    resume("throw", value);
  }
  function settle(f, v) {
    if (f(v), q.shift(), q.length)
      resume(q[0][0], q[0][1]);
  }
};
var __asyncDelegator = function(o) {
  var i, p;
  return i = {}, verb("next"), verb("throw", function(e) {
    throw e;
  }), verb("return"), i[Symbol.iterator] = function() {
    return this;
  }, i;
  function verb(n, f) {
    i[n] = o[n] ? function(v) {
      return (p = !p) ? { value: __await(o[n](v)), done: false } : f ? f(v) : v;
    } : f;
  }
};

// node_modules/@connectrpc/connect/dist/esm/promise-client.js
function createPromiseClient(service, transport) {
  return makeAnyClient(service, (method) => {
    switch (method.kind) {
      case MethodKind.Unary:
        return createUnaryFn(transport, service, method);
      case MethodKind.ServerStreaming:
        return createServerStreamingFn(transport, service, method);
      case MethodKind.ClientStreaming:
        return createClientStreamingFn(transport, service, method);
      case MethodKind.BiDiStreaming:
        return createBiDiStreamingFn(transport, service, method);
      default:
        return null;
    }
  });
}
function createUnaryFn(transport, service, method) {
  return async function(input, options) {
    var _a, _b;
    const response = await transport.unary(service, method, options === null || options === undefined ? undefined : options.signal, options === null || options === undefined ? undefined : options.timeoutMs, options === null || options === undefined ? undefined : options.headers, input, options === null || options === undefined ? undefined : options.contextValues);
    (_a = options === null || options === undefined ? undefined : options.onHeader) === null || _a === undefined || _a.call(options, response.header);
    (_b = options === null || options === undefined ? undefined : options.onTrailer) === null || _b === undefined || _b.call(options, response.trailer);
    return response.message;
  };
}
function createServerStreamingFn(transport, service, method) {
  return function(input, options) {
    return handleStreamResponse(transport.stream(service, method, options === null || options === undefined ? undefined : options.signal, options === null || options === undefined ? undefined : options.timeoutMs, options === null || options === undefined ? undefined : options.headers, createAsyncIterable([input]), options === null || options === undefined ? undefined : options.contextValues), options);
  };
}
function createClientStreamingFn(transport, service, method) {
  return async function(request, options) {
    var _a, e_1, _b, _c;
    var _d, _e;
    const response = await transport.stream(service, method, options === null || options === undefined ? undefined : options.signal, options === null || options === undefined ? undefined : options.timeoutMs, options === null || options === undefined ? undefined : options.headers, request, options === null || options === undefined ? undefined : options.contextValues);
    (_d = options === null || options === undefined ? undefined : options.onHeader) === null || _d === undefined || _d.call(options, response.header);
    let singleMessage;
    try {
      for (var _f = true, _g = __asyncValues2(response.message), _h;_h = await _g.next(), _a = _h.done, !_a; _f = true) {
        _c = _h.value;
        _f = false;
        const message3 = _c;
        singleMessage = message3;
      }
    } catch (e_1_1) {
      e_1 = { error: e_1_1 };
    } finally {
      try {
        if (!_f && !_a && (_b = _g.return))
          await _b.call(_g);
      } finally {
        if (e_1)
          throw e_1.error;
      }
    }
    if (!singleMessage) {
      throw new ConnectError("protocol error: missing response message", Code.Internal);
    }
    (_e = options === null || options === undefined ? undefined : options.onTrailer) === null || _e === undefined || _e.call(options, response.trailer);
    return singleMessage;
  };
}
function createBiDiStreamingFn(transport, service, method) {
  return function(request, options) {
    return handleStreamResponse(transport.stream(service, method, options === null || options === undefined ? undefined : options.signal, options === null || options === undefined ? undefined : options.timeoutMs, options === null || options === undefined ? undefined : options.headers, request, options === null || options === undefined ? undefined : options.contextValues), options);
  };
}
var handleStreamResponse = function(stream, options) {
  const it = function() {
    var _a, _b;
    return __asyncGenerator2(this, arguments, function* () {
      const response = yield __await2(stream);
      (_a = options === null || options === undefined ? undefined : options.onHeader) === null || _a === undefined || _a.call(options, response.header);
      yield __await2(yield* __asyncDelegator2(__asyncValues2(response.message)));
      (_b = options === null || options === undefined ? undefined : options.onTrailer) === null || _b === undefined || _b.call(options, response.trailer);
    });
  }()[Symbol.asyncIterator]();
  return {
    [Symbol.asyncIterator]: () => ({
      next: () => it.next()
    })
  };
};
var __asyncValues2 = function(o) {
  if (!Symbol.asyncIterator)
    throw new TypeError("Symbol.asyncIterator is not defined.");
  var m = o[Symbol.asyncIterator], i;
  return m ? m.call(o) : (o = typeof __values === "function" ? __values(o) : o[Symbol.iterator](), i = {}, verb("next"), verb("throw"), verb("return"), i[Symbol.asyncIterator] = function() {
    return this;
  }, i);
  function verb(n) {
    i[n] = o[n] && function(v) {
      return new Promise(function(resolve, reject) {
        v = o[n](v), settle(resolve, reject, v.done, v.value);
      });
    };
  }
  function settle(resolve, reject, d, v) {
    Promise.resolve(v).then(function(v2) {
      resolve({ value: v2, done: d });
    }, reject);
  }
};
var __await2 = function(v) {
  return this instanceof __await2 ? (this.v = v, this) : new __await2(v);
};
var __asyncDelegator2 = function(o) {
  var i, p;
  return i = {}, verb("next"), verb("throw", function(e) {
    throw e;
  }), verb("return"), i[Symbol.iterator] = function() {
    return this;
  }, i;
  function verb(n, f) {
    i[n] = o[n] ? function(v) {
      return (p = !p) ? { value: __await2(o[n](v)), done: false } : f ? f(v) : v;
    } : f;
  }
};
var __asyncGenerator2 = function(thisArg, _arguments, generator) {
  if (!Symbol.asyncIterator)
    throw new TypeError("Symbol.asyncIterator is not defined.");
  var g = generator.apply(thisArg, _arguments || []), i, q = [];
  return i = {}, verb("next"), verb("throw"), verb("return", awaitReturn), i[Symbol.asyncIterator] = function() {
    return this;
  }, i;
  function awaitReturn(f) {
    return function(v) {
      return Promise.resolve(v).then(f, reject);
    };
  }
  function verb(n, f) {
    if (g[n]) {
      i[n] = function(v) {
        return new Promise(function(a, b) {
          q.push([n, v, a, b]) > 1 || resume(n, v);
        });
      };
      if (f)
        i[n] = f(i[n]);
    }
  }
  function resume(n, v) {
    try {
      step(g[n](v));
    } catch (e) {
      settle(q[0][3], e);
    }
  }
  function step(r) {
    r.value instanceof __await2 ? Promise.resolve(r.value.v).then(fulfill, reject) : settle(q[0][2], r);
  }
  function fulfill(value) {
    resume("next", value);
  }
  function reject(value) {
    resume("throw", value);
  }
  function settle(f, v) {
    if (f(v), q.shift(), q.length)
      resume(q[0][0], q[0][1]);
  }
};
// node_modules/@connectrpc/connect/dist/esm/protocol/signals.js
function createLinkedAbortController(...signals) {
  const controller = new AbortController;
  const sa = signals.filter((s) => s !== undefined).concat(controller.signal);
  for (const signal of sa) {
    if (signal.aborted) {
      onAbort.apply(signal);
      break;
    }
    signal.addEventListener("abort", onAbort);
  }
  function onAbort() {
    if (!controller.signal.aborted) {
      controller.abort(getAbortSignalReason(this));
    }
    for (const signal of sa) {
      signal.removeEventListener("abort", onAbort);
    }
  }
  return controller;
}
function createDeadlineSignal(timeoutMs) {
  const controller = new AbortController;
  const listener = () => {
    controller.abort(new ConnectError("the operation timed out", Code.DeadlineExceeded));
  };
  let timeoutId;
  if (timeoutMs !== undefined) {
    if (timeoutMs <= 0)
      listener();
    else
      timeoutId = setTimeout(listener, timeoutMs);
  }
  return {
    signal: controller.signal,
    cleanup: () => clearTimeout(timeoutId)
  };
}
function getAbortSignalReason(signal) {
  if (!signal.aborted) {
    return;
  }
  if (signal.reason !== undefined) {
    return signal.reason;
  }
  const e = new Error("This operation was aborted");
  e.name = "AbortError";
  return e;
}

// node_modules/@connectrpc/connect/dist/esm/context-values.js
function createContextValues() {
  return {
    get(key) {
      return key.id in this ? this[key.id] : key.defaultValue;
    },
    set(key, value) {
      this[key.id] = value;
      return this;
    },
    delete(key) {
      delete this[key.id];
      return this;
    }
  };
}

// node_modules/@connectrpc/connect/dist/esm/protocol/create-method-url.js
function createMethodUrl(baseUrl, service, method) {
  const s = typeof service == "string" ? service : service.typeName;
  const m = typeof method == "string" ? method : method.name;
  return baseUrl.toString().replace(/\/?$/, `/${s}/${m}`);
}

// node_modules/@connectrpc/connect/dist/esm/protocol/normalize.js
function normalize(type, message3) {
  return message3 instanceof type ? message3 : new type(message3);
}
function normalizeIterable(messageType, input) {
  function transform(result) {
    if (result.done === true) {
      return result;
    }
    return {
      done: result.done,
      value: normalize(messageType, result.value)
    };
  }
  return {
    [Symbol.asyncIterator]() {
      const it = input[Symbol.asyncIterator]();
      const res = {
        next: () => it.next().then(transform)
      };
      if (it.throw !== undefined) {
        res.throw = (e) => it.throw(e).then(transform);
      }
      if (it.return !== undefined) {
        res.return = (v) => it.return(v).then(transform);
      }
      return res;
    }
  };
}

// node_modules/@connectrpc/connect/dist/esm/interceptor.js
function applyInterceptors(next, interceptors) {
  var _a;
  return (_a = interceptors === null || interceptors === undefined ? undefined : interceptors.concat().reverse().reduce((n, i) => i(n), next)) !== null && _a !== undefined ? _a : next;
}

// node_modules/@connectrpc/connect/dist/esm/protocol/serialization.js
function getJsonOptions(options) {
  var _a;
  const o = Object.assign({}, options);
  (_a = o.ignoreUnknownFields) !== null && _a !== undefined || (o.ignoreUnknownFields = true);
  return o;
}
function createClientMethodSerializers(method, useBinaryFormat, jsonOptions, binaryOptions) {
  const input = useBinaryFormat ? createBinarySerialization(method.I, binaryOptions) : createJsonSerialization(method.I, jsonOptions);
  const output = useBinaryFormat ? createBinarySerialization(method.O, binaryOptions) : createJsonSerialization(method.O, jsonOptions);
  return { parse: output.parse, serialize: input.serialize };
}
function createBinarySerialization(messageType, options) {
  return {
    parse(data) {
      try {
        return messageType.fromBinary(data, options);
      } catch (e) {
        const m = e instanceof Error ? e.message : String(e);
        throw new ConnectError(`parse binary: ${m}`, Code.InvalidArgument);
      }
    },
    serialize(data) {
      try {
        return data.toBinary(options);
      } catch (e) {
        const m = e instanceof Error ? e.message : String(e);
        throw new ConnectError(`serialize binary: ${m}`, Code.Internal);
      }
    }
  };
}
function createJsonSerialization(messageType, options) {
  var _a, _b;
  const textEncoder = (_a = options === null || options === undefined ? undefined : options.textEncoder) !== null && _a !== undefined ? _a : new TextEncoder;
  const textDecoder = (_b = options === null || options === undefined ? undefined : options.textDecoder) !== null && _b !== undefined ? _b : new TextDecoder;
  const o = getJsonOptions(options);
  return {
    parse(data) {
      try {
        const json = textDecoder.decode(data);
        return messageType.fromJsonString(json, o);
      } catch (e) {
        throw ConnectError.from(e, Code.InvalidArgument);
      }
    },
    serialize(data) {
      try {
        const json = data.toJsonString(o);
        return textEncoder.encode(json);
      } catch (e) {
        throw ConnectError.from(e, Code.Internal);
      }
    }
  };
}

// node_modules/@connectrpc/connect/dist/esm/protocol-connect/content-type.js
function parseContentType(contentType) {
  const match = contentType === null || contentType === undefined ? undefined : contentType.match(contentTypeRegExp);
  if (!match) {
    return;
  }
  const stream = !!match[1];
  const binary = !!match[3];
  return { stream, binary };
}
var contentTypeRegExp = /^application\/(connect\+)?(?:(json)(?:; ?charset=utf-?8)?|(proto))$/i;
var contentTypeUnaryProto = "application/proto";
var contentTypeUnaryJson = "application/json";
var contentTypeStreamProto = "application/connect+proto";
var contentTypeStreamJson = "application/connect+json";

// node_modules/@connectrpc/connect/dist/esm/protocol-connect/error-json.js
function errorFromJson(jsonValue, metadata, fallback2) {
  if (metadata) {
    new Headers(metadata).forEach((value, key) => fallback2.metadata.append(key, value));
  }
  if (typeof jsonValue !== "object" || jsonValue == null || Array.isArray(jsonValue) || !("code" in jsonValue) || typeof jsonValue.code !== "string") {
    throw fallback2;
  }
  const code7 = codeFromString(jsonValue.code);
  if (code7 === undefined) {
    throw fallback2;
  }
  const message3 = jsonValue.message;
  if (message3 != null && typeof message3 !== "string") {
    throw fallback2;
  }
  const error = new ConnectError(message3 !== null && message3 !== undefined ? message3 : "", code7, metadata);
  if ("details" in jsonValue && Array.isArray(jsonValue.details)) {
    for (const detail of jsonValue.details) {
      if (detail === null || typeof detail != "object" || Array.isArray(detail) || typeof detail.type != "string" || typeof detail.value != "string" || "debug" in detail && typeof detail.debug != "object") {
        throw fallback2;
      }
      try {
        error.details.push({
          type: detail.type,
          value: protoBase64.dec(detail.value),
          debug: detail.debug
        });
      } catch (e) {
        throw fallback2;
      }
    }
  }
  return error;
}

// node_modules/@connectrpc/connect/dist/esm/protocol-connect/end-stream.js
function endStreamFromJson(data) {
  const parseErr = new ConnectError("invalid end stream", Code.InvalidArgument);
  let jsonValue;
  try {
    jsonValue = JSON.parse(typeof data == "string" ? data : new TextDecoder().decode(data));
  } catch (e) {
    throw parseErr;
  }
  if (typeof jsonValue != "object" || jsonValue == null || Array.isArray(jsonValue)) {
    throw parseErr;
  }
  const metadata = new Headers;
  if ("metadata" in jsonValue) {
    if (typeof jsonValue.metadata != "object" || jsonValue.metadata == null || Array.isArray(jsonValue.metadata)) {
      throw parseErr;
    }
    for (const [key, values] of Object.entries(jsonValue.metadata)) {
      if (!Array.isArray(values) || values.some((value) => typeof value != "string")) {
        throw parseErr;
      }
      for (const value of values) {
        metadata.append(key, value);
      }
    }
  }
  const error = "error" in jsonValue ? errorFromJson(jsonValue.error, metadata, parseErr) : undefined;
  return { metadata, error };
}
var endStreamFlag = 2;

// node_modules/@connectrpc/connect/dist/esm/protocol-connect/headers.js
var headerContentType = "Content-Type";
var headerUnaryContentLength = "Content-Length";
var headerUnaryEncoding = "Content-Encoding";
var headerUnaryAcceptEncoding = "Accept-Encoding";
var headerTimeout = "Connect-Timeout-Ms";
var headerProtocolVersion = "Connect-Protocol-Version";
var headerUserAgent = "User-Agent";

// node_modules/@connectrpc/connect/dist/esm/protocol-connect/http-status.js
function codeFromHttpStatus(httpStatus) {
  switch (httpStatus) {
    case 400:
      return Code.InvalidArgument;
    case 401:
      return Code.Unauthenticated;
    case 403:
      return Code.PermissionDenied;
    case 404:
      return Code.Unimplemented;
    case 408:
      return Code.DeadlineExceeded;
    case 409:
      return Code.Aborted;
    case 412:
      return Code.FailedPrecondition;
    case 413:
      return Code.ResourceExhausted;
    case 415:
      return Code.Internal;
    case 429:
      return Code.Unavailable;
    case 431:
      return Code.ResourceExhausted;
    case 502:
      return Code.Unavailable;
    case 503:
      return Code.Unavailable;
    case 504:
      return Code.Unavailable;
    default:
      return Code.Unknown;
  }
}

// node_modules/@connectrpc/connect/dist/esm/protocol-connect/trailer-mux.js
function trailerDemux(header) {
  const h = new Headers, t = new Headers;
  header.forEach((value, key) => {
    if (key.toLowerCase().startsWith("trailer-")) {
      t.set(key.substring(8), value);
    } else {
      h.set(key, value);
    }
  });
  return [h, t];
}

// node_modules/@connectrpc/connect/dist/esm/protocol-connect/version.js
var protocolVersion = "1";
// node_modules/@connectrpc/connect/dist/esm/protocol-connect/request-header.js
function requestHeader(methodKind, useBinaryFormat, timeoutMs, userProvidedHeaders, setUserAgent) {
  const result = new Headers(userProvidedHeaders !== null && userProvidedHeaders !== undefined ? userProvidedHeaders : {});
  if (timeoutMs !== undefined) {
    result.set(headerTimeout, `${timeoutMs}`);
  }
  result.set(headerContentType, methodKind == MethodKind.Unary ? useBinaryFormat ? contentTypeUnaryProto : contentTypeUnaryJson : useBinaryFormat ? contentTypeStreamProto : contentTypeStreamJson);
  result.set(headerProtocolVersion, protocolVersion);
  if (setUserAgent) {
    result.set(headerUserAgent, "connect-es/1.4.0");
  }
  return result;
}

// node_modules/@connectrpc/connect/dist/esm/protocol-connect/validate-response.js
function validateResponse(methodKind, status, headers2) {
  const mimeType = headers2.get("Content-Type");
  const parsedType = parseContentType(mimeType);
  if (status !== 200) {
    const errorFromStatus = new ConnectError(`HTTP ${status}`, codeFromHttpStatus(status), headers2);
    if (methodKind == MethodKind.Unary && parsedType && !parsedType.binary) {
      return { isUnaryError: true, unaryError: errorFromStatus };
    }
    throw errorFromStatus;
  }
  return { isUnaryError: false };
}

// node_modules/@connectrpc/connect/dist/esm/protocol-connect/get-request.js
var encodeMessageForUrl = function(message3, useBase64) {
  if (useBase64) {
    return protoBase64.enc(message3).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
  } else {
    return encodeURIComponent(new TextDecoder().decode(message3));
  }
};
function transformConnectPostToGetRequest(request, message3, useBase64) {
  let query = `?connect=v${protocolVersion}`;
  const contentType = request.header.get(headerContentType);
  if ((contentType === null || contentType === undefined ? undefined : contentType.indexOf(contentTypePrefix)) === 0) {
    query += "&encoding=" + encodeURIComponent(contentType.slice(contentTypePrefix.length));
  }
  const compression = request.header.get(headerUnaryEncoding);
  if (compression !== null && compression !== "identity") {
    query += "&compression=" + encodeURIComponent(compression);
    useBase64 = true;
  }
  if (useBase64) {
    query += "&base64=1";
  }
  query += "&message=" + encodeMessageForUrl(message3, useBase64);
  const url = request.url + query;
  const header = new Headers(request.header);
  [
    headerProtocolVersion,
    headerContentType,
    headerUnaryContentLength,
    headerUnaryEncoding,
    headerUnaryAcceptEncoding
  ].forEach((h) => header.delete(h));
  return Object.assign(Object.assign({}, request), {
    init: Object.assign(Object.assign({}, request.init), { method: "GET" }),
    url,
    header
  });
}
var contentTypePrefix = "application/";

// node_modules/@connectrpc/connect/dist/esm/protocol/run-call.js
function runUnaryCall(opt) {
  const next = applyInterceptors(opt.next, opt.interceptors);
  const [signal, abort, done] = setupSignal(opt);
  const req = Object.assign(Object.assign({}, opt.req), { message: normalize(opt.req.method.I, opt.req.message), signal });
  return next(req).then((res) => {
    done();
    return res;
  }, abort);
}
function runStreamingCall(opt) {
  const next = applyInterceptors(opt.next, opt.interceptors);
  const [signal, abort, done] = setupSignal(opt);
  const req = Object.assign(Object.assign({}, opt.req), { message: normalizeIterable(opt.req.method.I, opt.req.message), signal });
  let doneCalled = false;
  signal.addEventListener("abort", function() {
    var _a, _b;
    const it = opt.req.message[Symbol.asyncIterator]();
    if (!doneCalled) {
      (_a = it.throw) === null || _a === undefined || _a.call(it, this.reason).catch(() => {
      });
    }
    (_b = it.return) === null || _b === undefined || _b.call(it).catch(() => {
    });
  });
  return next(req).then((res) => {
    return Object.assign(Object.assign({}, res), { message: {
      [Symbol.asyncIterator]() {
        const it = res.message[Symbol.asyncIterator]();
        return {
          next() {
            return it.next().then((r) => {
              if (r.done == true) {
                doneCalled = true;
                done();
              }
              return r;
            }, abort);
          }
        };
      }
    } });
  }, abort);
}
var setupSignal = function(opt) {
  const { signal, cleanup } = createDeadlineSignal(opt.timeoutMs);
  const controller = createLinkedAbortController(opt.signal, signal);
  return [
    controller.signal,
    function abort(reason) {
      const e = ConnectError.from(signal.aborted ? getAbortSignalReason(signal) : reason);
      controller.abort(e);
      cleanup();
      return Promise.reject(e);
    },
    function done() {
      cleanup();
      controller.abort();
    }
  ];
};
// node_modules/@connectrpc/connect-web/dist/esm/assert-fetch-api.js
function assertFetchApi() {
  try {
    new Headers;
  } catch (_) {
    throw new Error("connect-web requires the fetch API. Are you running on an old version of Node.js? Node.js is not supported in Connect for Web - please stay tuned for Connect for Node.");
  }
}

// node_modules/@connectrpc/connect-web/dist/esm/connect-transport.js
function createConnectTransport(options) {
  var _a;
  assertFetchApi();
  const useBinaryFormat = (_a = options.useBinaryFormat) !== null && _a !== undefined ? _a : false;
  return {
    async unary(service, method, signal, timeoutMs, header, message3, contextValues) {
      var _a2;
      const { serialize, parse } = createClientMethodSerializers(method, useBinaryFormat, options.jsonOptions, options.binaryOptions);
      timeoutMs = timeoutMs === undefined ? options.defaultTimeoutMs : timeoutMs <= 0 ? undefined : timeoutMs;
      return await runUnaryCall({
        interceptors: options.interceptors,
        signal,
        timeoutMs,
        req: {
          stream: false,
          service,
          method,
          url: createMethodUrl(options.baseUrl, service, method),
          init: {
            method: "POST",
            credentials: (_a2 = options.credentials) !== null && _a2 !== undefined ? _a2 : "same-origin",
            redirect: "error",
            mode: "cors"
          },
          header: requestHeader(method.kind, useBinaryFormat, timeoutMs, header, false),
          contextValues: contextValues !== null && contextValues !== undefined ? contextValues : createContextValues(),
          message: message3
        },
        next: async (req) => {
          var _a3;
          const useGet = options.useHttpGet === true && method.idempotency === MethodIdempotency.NoSideEffects;
          let body = null;
          if (useGet) {
            req = transformConnectPostToGetRequest(req, serialize(req.message), useBinaryFormat);
          } else {
            body = serialize(req.message);
          }
          const fetch = (_a3 = options.fetch) !== null && _a3 !== undefined ? _a3 : globalThis.fetch;
          const response = await fetch(req.url, Object.assign(Object.assign({}, req.init), { headers: req.header, signal: req.signal, body }));
          const { isUnaryError, unaryError } = validateResponse(method.kind, response.status, response.headers);
          if (isUnaryError) {
            throw errorFromJson(await response.json(), appendHeaders(...trailerDemux(response.headers)), unaryError);
          }
          const [demuxedHeader, demuxedTrailer] = trailerDemux(response.headers);
          return {
            stream: false,
            service,
            method,
            header: demuxedHeader,
            message: useBinaryFormat ? parse(new Uint8Array(await response.arrayBuffer())) : method.O.fromJson(await response.json(), getJsonOptions(options.jsonOptions)),
            trailer: demuxedTrailer
          };
        }
      });
    },
    async stream(service, method, signal, timeoutMs, header, input, contextValues) {
      var _a2;
      const { serialize, parse } = createClientMethodSerializers(method, useBinaryFormat, options.jsonOptions, options.binaryOptions);
      function parseResponseBody(body, trailerTarget, header2) {
        return __asyncGenerator3(this, arguments, function* parseResponseBody_1() {
          const reader = createEnvelopeReadableStream(body).getReader();
          let endStreamReceived = false;
          for (;; ) {
            const result = yield __await3(reader.read());
            if (result.done) {
              break;
            }
            const { flags, data } = result.value;
            if ((flags & endStreamFlag) === endStreamFlag) {
              endStreamReceived = true;
              const endStream = endStreamFromJson(data);
              if (endStream.error) {
                const error = endStream.error;
                header2.forEach((value, key) => {
                  error.metadata.append(key, value);
                });
                throw error;
              }
              endStream.metadata.forEach((value, key) => trailerTarget.set(key, value));
              continue;
            }
            yield yield __await3(parse(data));
          }
          if (!endStreamReceived) {
            throw "missing EndStreamResponse";
          }
        });
      }
      async function createRequestBody(input2) {
        if (method.kind != MethodKind.ServerStreaming) {
          throw "The fetch API does not support streaming request bodies";
        }
        const r = await input2[Symbol.asyncIterator]().next();
        if (r.done == true) {
          throw "missing request message";
        }
        return encodeEnvelope(0, serialize(r.value));
      }
      timeoutMs = timeoutMs === undefined ? options.defaultTimeoutMs : timeoutMs <= 0 ? undefined : timeoutMs;
      return await runStreamingCall({
        interceptors: options.interceptors,
        timeoutMs,
        signal,
        req: {
          stream: true,
          service,
          method,
          url: createMethodUrl(options.baseUrl, service, method),
          init: {
            method: "POST",
            credentials: (_a2 = options.credentials) !== null && _a2 !== undefined ? _a2 : "same-origin",
            redirect: "error",
            mode: "cors"
          },
          header: requestHeader(method.kind, useBinaryFormat, timeoutMs, header, false),
          contextValues: contextValues !== null && contextValues !== undefined ? contextValues : createContextValues(),
          message: input
        },
        next: async (req) => {
          var _a3;
          const fetch = (_a3 = options.fetch) !== null && _a3 !== undefined ? _a3 : globalThis.fetch;
          const fRes = await fetch(req.url, Object.assign(Object.assign({}, req.init), { headers: req.header, signal: req.signal, body: await createRequestBody(req.message) }));
          validateResponse(method.kind, fRes.status, fRes.headers);
          if (fRes.body === null) {
            throw "missing response body";
          }
          const trailer = new Headers;
          const res = Object.assign(Object.assign({}, req), { header: fRes.headers, trailer, message: parseResponseBody(fRes.body, trailer, fRes.headers) });
          return res;
        }
      });
    }
  };
}
var __await3 = function(v) {
  return this instanceof __await3 ? (this.v = v, this) : new __await3(v);
};
var __asyncGenerator3 = function(thisArg, _arguments, generator) {
  if (!Symbol.asyncIterator)
    throw new TypeError("Symbol.asyncIterator is not defined.");
  var g = generator.apply(thisArg, _arguments || []), i, q = [];
  return i = {}, verb("next"), verb("throw"), verb("return", awaitReturn), i[Symbol.asyncIterator] = function() {
    return this;
  }, i;
  function awaitReturn(f) {
    return function(v) {
      return Promise.resolve(v).then(f, reject);
    };
  }
  function verb(n, f) {
    if (g[n]) {
      i[n] = function(v) {
        return new Promise(function(a, b) {
          q.push([n, v, a, b]) > 1 || resume(n, v);
        });
      };
      if (f)
        i[n] = f(i[n]);
    }
  }
  function resume(n, v) {
    try {
      step(g[n](v));
    } catch (e) {
      settle(q[0][3], e);
    }
  }
  function step(r) {
    r.value instanceof __await3 ? Promise.resolve(r.value.v).then(fulfill, reject) : settle(q[0][2], r);
  }
  function fulfill(value) {
    resume("next", value);
  }
  function reject(value) {
    resume("throw", value);
  }
  function settle(f, v) {
    if (f(v), q.shift(), q.length)
      resume(q[0][0], q[0][1]);
  }
};
// js/api/v1/ipam_pb.ts
class Prefix extends Message {
  cidr = "";
  parentCidr = "";
  constructor(data) {
    super();
    proto3.util.initPartial(data, this);
  }
  static runtime = proto3;
  static typeName = "api.v1.Prefix";
  static fields = proto3.util.newFieldList(() => [
    { no: 1, name: "cidr", kind: "scalar", T: 9 },
    { no: 2, name: "parent_cidr", kind: "scalar", T: 9 }
  ]);
  static fromBinary(bytes, options) {
    return new Prefix().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new Prefix().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new Prefix().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(Prefix, a, b);
  }
}

class CreatePrefixResponse extends Message {
  prefix;
  constructor(data) {
    super();
    proto3.util.initPartial(data, this);
  }
  static runtime = proto3;
  static typeName = "api.v1.CreatePrefixResponse";
  static fields = proto3.util.newFieldList(() => [
    { no: 1, name: "prefix", kind: "message", T: Prefix }
  ]);
  static fromBinary(bytes, options) {
    return new CreatePrefixResponse().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new CreatePrefixResponse().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new CreatePrefixResponse().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(CreatePrefixResponse, a, b);
  }
}

class DeletePrefixResponse extends Message {
  prefix;
  constructor(data) {
    super();
    proto3.util.initPartial(data, this);
  }
  static runtime = proto3;
  static typeName = "api.v1.DeletePrefixResponse";
  static fields = proto3.util.newFieldList(() => [
    { no: 1, name: "prefix", kind: "message", T: Prefix }
  ]);
  static fromBinary(bytes, options) {
    return new DeletePrefixResponse().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new DeletePrefixResponse().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new DeletePrefixResponse().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(DeletePrefixResponse, a, b);
  }
}

class GetPrefixResponse extends Message {
  prefix;
  constructor(data) {
    super();
    proto3.util.initPartial(data, this);
  }
  static runtime = proto3;
  static typeName = "api.v1.GetPrefixResponse";
  static fields = proto3.util.newFieldList(() => [
    { no: 1, name: "prefix", kind: "message", T: Prefix }
  ]);
  static fromBinary(bytes, options) {
    return new GetPrefixResponse().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new GetPrefixResponse().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new GetPrefixResponse().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(GetPrefixResponse, a, b);
  }
}

class AcquireChildPrefixResponse extends Message {
  prefix;
  constructor(data) {
    super();
    proto3.util.initPartial(data, this);
  }
  static runtime = proto3;
  static typeName = "api.v1.AcquireChildPrefixResponse";
  static fields = proto3.util.newFieldList(() => [
    { no: 1, name: "prefix", kind: "message", T: Prefix }
  ]);
  static fromBinary(bytes, options) {
    return new AcquireChildPrefixResponse().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new AcquireChildPrefixResponse().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new AcquireChildPrefixResponse().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(AcquireChildPrefixResponse, a, b);
  }
}

class ReleaseChildPrefixResponse extends Message {
  prefix;
  constructor(data) {
    super();
    proto3.util.initPartial(data, this);
  }
  static runtime = proto3;
  static typeName = "api.v1.ReleaseChildPrefixResponse";
  static fields = proto3.util.newFieldList(() => [
    { no: 1, name: "prefix", kind: "message", T: Prefix }
  ]);
  static fromBinary(bytes, options) {
    return new ReleaseChildPrefixResponse().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new ReleaseChildPrefixResponse().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new ReleaseChildPrefixResponse().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(ReleaseChildPrefixResponse, a, b);
  }
}

class CreatePrefixRequest extends Message {
  cidr = "";
  namespace;
  constructor(data) {
    super();
    proto3.util.initPartial(data, this);
  }
  static runtime = proto3;
  static typeName = "api.v1.CreatePrefixRequest";
  static fields = proto3.util.newFieldList(() => [
    { no: 1, name: "cidr", kind: "scalar", T: 9 },
    { no: 2, name: "namespace", kind: "scalar", T: 9, opt: true }
  ]);
  static fromBinary(bytes, options) {
    return new CreatePrefixRequest().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new CreatePrefixRequest().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new CreatePrefixRequest().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(CreatePrefixRequest, a, b);
  }
}

class DeletePrefixRequest extends Message {
  cidr = "";
  namespace;
  constructor(data) {
    super();
    proto3.util.initPartial(data, this);
  }
  static runtime = proto3;
  static typeName = "api.v1.DeletePrefixRequest";
  static fields = proto3.util.newFieldList(() => [
    { no: 1, name: "cidr", kind: "scalar", T: 9 },
    { no: 2, name: "namespace", kind: "scalar", T: 9, opt: true }
  ]);
  static fromBinary(bytes, options) {
    return new DeletePrefixRequest().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new DeletePrefixRequest().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new DeletePrefixRequest().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(DeletePrefixRequest, a, b);
  }
}

class GetPrefixRequest extends Message {
  cidr = "";
  namespace;
  constructor(data) {
    super();
    proto3.util.initPartial(data, this);
  }
  static runtime = proto3;
  static typeName = "api.v1.GetPrefixRequest";
  static fields = proto3.util.newFieldList(() => [
    { no: 1, name: "cidr", kind: "scalar", T: 9 },
    { no: 2, name: "namespace", kind: "scalar", T: 9, opt: true }
  ]);
  static fromBinary(bytes, options) {
    return new GetPrefixRequest().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new GetPrefixRequest().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new GetPrefixRequest().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(GetPrefixRequest, a, b);
  }
}

class ListPrefixesRequest extends Message {
  namespace;
  constructor(data) {
    super();
    proto3.util.initPartial(data, this);
  }
  static runtime = proto3;
  static typeName = "api.v1.ListPrefixesRequest";
  static fields = proto3.util.newFieldList(() => [
    { no: 1, name: "namespace", kind: "scalar", T: 9, opt: true }
  ]);
  static fromBinary(bytes, options) {
    return new ListPrefixesRequest().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new ListPrefixesRequest().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new ListPrefixesRequest().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(ListPrefixesRequest, a, b);
  }
}

class ListPrefixesResponse extends Message {
  prefixes = [];
  constructor(data) {
    super();
    proto3.util.initPartial(data, this);
  }
  static runtime = proto3;
  static typeName = "api.v1.ListPrefixesResponse";
  static fields = proto3.util.newFieldList(() => [
    { no: 1, name: "prefixes", kind: "message", T: Prefix, repeated: true }
  ]);
  static fromBinary(bytes, options) {
    return new ListPrefixesResponse().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new ListPrefixesResponse().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new ListPrefixesResponse().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(ListPrefixesResponse, a, b);
  }
}

class PrefixUsageRequest extends Message {
  cidr = "";
  namespace;
  constructor(data) {
    super();
    proto3.util.initPartial(data, this);
  }
  static runtime = proto3;
  static typeName = "api.v1.PrefixUsageRequest";
  static fields = proto3.util.newFieldList(() => [
    { no: 1, name: "cidr", kind: "scalar", T: 9 },
    { no: 2, name: "namespace", kind: "scalar", T: 9, opt: true }
  ]);
  static fromBinary(bytes, options) {
    return new PrefixUsageRequest().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new PrefixUsageRequest().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new PrefixUsageRequest().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(PrefixUsageRequest, a, b);
  }
}

class PrefixUsageResponse extends Message {
  availableIps = protoInt64.zero;
  acquiredIps = protoInt64.zero;
  availableSmallestPrefixes = protoInt64.zero;
  availablePrefixes = [];
  acquiredPrefixes = protoInt64.zero;
  constructor(data) {
    super();
    proto3.util.initPartial(data, this);
  }
  static runtime = proto3;
  static typeName = "api.v1.PrefixUsageResponse";
  static fields = proto3.util.newFieldList(() => [
    { no: 1, name: "available_ips", kind: "scalar", T: 4 },
    { no: 2, name: "acquired_ips", kind: "scalar", T: 4 },
    { no: 3, name: "available_smallest_prefixes", kind: "scalar", T: 4 },
    { no: 4, name: "available_prefixes", kind: "scalar", T: 9, repeated: true },
    { no: 5, name: "acquired_prefixes", kind: "scalar", T: 4 }
  ]);
  static fromBinary(bytes, options) {
    return new PrefixUsageResponse().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new PrefixUsageResponse().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new PrefixUsageResponse().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(PrefixUsageResponse, a, b);
  }
}

class AcquireChildPrefixRequest extends Message {
  cidr = "";
  length = 0;
  childCidr;
  namespace;
  constructor(data) {
    super();
    proto3.util.initPartial(data, this);
  }
  static runtime = proto3;
  static typeName = "api.v1.AcquireChildPrefixRequest";
  static fields = proto3.util.newFieldList(() => [
    { no: 1, name: "cidr", kind: "scalar", T: 9 },
    { no: 2, name: "length", kind: "scalar", T: 13 },
    { no: 3, name: "child_cidr", kind: "scalar", T: 9, opt: true },
    { no: 4, name: "namespace", kind: "scalar", T: 9, opt: true }
  ]);
  static fromBinary(bytes, options) {
    return new AcquireChildPrefixRequest().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new AcquireChildPrefixRequest().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new AcquireChildPrefixRequest().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(AcquireChildPrefixRequest, a, b);
  }
}

class ReleaseChildPrefixRequest extends Message {
  cidr = "";
  namespace;
  constructor(data) {
    super();
    proto3.util.initPartial(data, this);
  }
  static runtime = proto3;
  static typeName = "api.v1.ReleaseChildPrefixRequest";
  static fields = proto3.util.newFieldList(() => [
    { no: 1, name: "cidr", kind: "scalar", T: 9 },
    { no: 2, name: "namespace", kind: "scalar", T: 9, opt: true }
  ]);
  static fromBinary(bytes, options) {
    return new ReleaseChildPrefixRequest().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new ReleaseChildPrefixRequest().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new ReleaseChildPrefixRequest().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(ReleaseChildPrefixRequest, a, b);
  }
}

class IP extends Message {
  ip = "";
  parentPrefix = "";
  constructor(data) {
    super();
    proto3.util.initPartial(data, this);
  }
  static runtime = proto3;
  static typeName = "api.v1.IP";
  static fields = proto3.util.newFieldList(() => [
    { no: 1, name: "ip", kind: "scalar", T: 9 },
    { no: 2, name: "parent_prefix", kind: "scalar", T: 9 }
  ]);
  static fromBinary(bytes, options) {
    return new IP().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new IP().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new IP().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(IP, a, b);
  }
}

class AcquireIPResponse extends Message {
  ip;
  namespace;
  constructor(data) {
    super();
    proto3.util.initPartial(data, this);
  }
  static runtime = proto3;
  static typeName = "api.v1.AcquireIPResponse";
  static fields = proto3.util.newFieldList(() => [
    { no: 1, name: "ip", kind: "message", T: IP },
    { no: 2, name: "namespace", kind: "scalar", T: 9, opt: true }
  ]);
  static fromBinary(bytes, options) {
    return new AcquireIPResponse().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new AcquireIPResponse().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new AcquireIPResponse().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(AcquireIPResponse, a, b);
  }
}

class ReleaseIPResponse extends Message {
  ip;
  constructor(data) {
    super();
    proto3.util.initPartial(data, this);
  }
  static runtime = proto3;
  static typeName = "api.v1.ReleaseIPResponse";
  static fields = proto3.util.newFieldList(() => [
    { no: 1, name: "ip", kind: "message", T: IP }
  ]);
  static fromBinary(bytes, options) {
    return new ReleaseIPResponse().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new ReleaseIPResponse().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new ReleaseIPResponse().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(ReleaseIPResponse, a, b);
  }
}

class AcquireIPRequest extends Message {
  prefixCidr = "";
  ip;
  namespace;
  constructor(data) {
    super();
    proto3.util.initPartial(data, this);
  }
  static runtime = proto3;
  static typeName = "api.v1.AcquireIPRequest";
  static fields = proto3.util.newFieldList(() => [
    { no: 1, name: "prefix_cidr", kind: "scalar", T: 9 },
    { no: 2, name: "ip", kind: "scalar", T: 9, opt: true },
    { no: 3, name: "namespace", kind: "scalar", T: 9, opt: true }
  ]);
  static fromBinary(bytes, options) {
    return new AcquireIPRequest().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new AcquireIPRequest().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new AcquireIPRequest().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(AcquireIPRequest, a, b);
  }
}

class ReleaseIPRequest extends Message {
  prefixCidr = "";
  ip = "";
  namespace;
  constructor(data) {
    super();
    proto3.util.initPartial(data, this);
  }
  static runtime = proto3;
  static typeName = "api.v1.ReleaseIPRequest";
  static fields = proto3.util.newFieldList(() => [
    { no: 1, name: "prefix_cidr", kind: "scalar", T: 9 },
    { no: 2, name: "ip", kind: "scalar", T: 9 },
    { no: 3, name: "namespace", kind: "scalar", T: 9, opt: true }
  ]);
  static fromBinary(bytes, options) {
    return new ReleaseIPRequest().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new ReleaseIPRequest().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new ReleaseIPRequest().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(ReleaseIPRequest, a, b);
  }
}

class DumpRequest extends Message {
  namespace;
  constructor(data) {
    super();
    proto3.util.initPartial(data, this);
  }
  static runtime = proto3;
  static typeName = "api.v1.DumpRequest";
  static fields = proto3.util.newFieldList(() => [
    { no: 1, name: "namespace", kind: "scalar", T: 9, opt: true }
  ]);
  static fromBinary(bytes, options) {
    return new DumpRequest().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new DumpRequest().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new DumpRequest().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(DumpRequest, a, b);
  }
}

class DumpResponse extends Message {
  dump = "";
  constructor(data) {
    super();
    proto3.util.initPartial(data, this);
  }
  static runtime = proto3;
  static typeName = "api.v1.DumpResponse";
  static fields = proto3.util.newFieldList(() => [
    { no: 1, name: "dump", kind: "scalar", T: 9 }
  ]);
  static fromBinary(bytes, options) {
    return new DumpResponse().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new DumpResponse().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new DumpResponse().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(DumpResponse, a, b);
  }
}

class LoadRequest extends Message {
  dump = "";
  namespace;
  constructor(data) {
    super();
    proto3.util.initPartial(data, this);
  }
  static runtime = proto3;
  static typeName = "api.v1.LoadRequest";
  static fields = proto3.util.newFieldList(() => [
    { no: 1, name: "dump", kind: "scalar", T: 9 },
    { no: 2, name: "namespace", kind: "scalar", T: 9, opt: true }
  ]);
  static fromBinary(bytes, options) {
    return new LoadRequest().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new LoadRequest().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new LoadRequest().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(LoadRequest, a, b);
  }
}

class LoadResponse extends Message {
  constructor(data) {
    super();
    proto3.util.initPartial(data, this);
  }
  static runtime = proto3;
  static typeName = "api.v1.LoadResponse";
  static fields = proto3.util.newFieldList(() => []);
  static fromBinary(bytes, options) {
    return new LoadResponse().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new LoadResponse().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new LoadResponse().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(LoadResponse, a, b);
  }
}

class CreateNamespaceRequest extends Message {
  namespace = "";
  constructor(data) {
    super();
    proto3.util.initPartial(data, this);
  }
  static runtime = proto3;
  static typeName = "api.v1.CreateNamespaceRequest";
  static fields = proto3.util.newFieldList(() => [
    { no: 1, name: "namespace", kind: "scalar", T: 9 }
  ]);
  static fromBinary(bytes, options) {
    return new CreateNamespaceRequest().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new CreateNamespaceRequest().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new CreateNamespaceRequest().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(CreateNamespaceRequest, a, b);
  }
}

class CreateNamespaceResponse extends Message {
  constructor(data) {
    super();
    proto3.util.initPartial(data, this);
  }
  static runtime = proto3;
  static typeName = "api.v1.CreateNamespaceResponse";
  static fields = proto3.util.newFieldList(() => []);
  static fromBinary(bytes, options) {
    return new CreateNamespaceResponse().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new CreateNamespaceResponse().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new CreateNamespaceResponse().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(CreateNamespaceResponse, a, b);
  }
}

class ListNamespacesRequest extends Message {
  constructor(data) {
    super();
    proto3.util.initPartial(data, this);
  }
  static runtime = proto3;
  static typeName = "api.v1.ListNamespacesRequest";
  static fields = proto3.util.newFieldList(() => []);
  static fromBinary(bytes, options) {
    return new ListNamespacesRequest().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new ListNamespacesRequest().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new ListNamespacesRequest().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(ListNamespacesRequest, a, b);
  }
}

class ListNamespacesResponse extends Message {
  namespace = [];
  constructor(data) {
    super();
    proto3.util.initPartial(data, this);
  }
  static runtime = proto3;
  static typeName = "api.v1.ListNamespacesResponse";
  static fields = proto3.util.newFieldList(() => [
    { no: 1, name: "namespace", kind: "scalar", T: 9, repeated: true }
  ]);
  static fromBinary(bytes, options) {
    return new ListNamespacesResponse().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new ListNamespacesResponse().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new ListNamespacesResponse().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(ListNamespacesResponse, a, b);
  }
}

class DeleteNamespaceRequest extends Message {
  namespace = "";
  constructor(data) {
    super();
    proto3.util.initPartial(data, this);
  }
  static runtime = proto3;
  static typeName = "api.v1.DeleteNamespaceRequest";
  static fields = proto3.util.newFieldList(() => [
    { no: 1, name: "namespace", kind: "scalar", T: 9 }
  ]);
  static fromBinary(bytes, options) {
    return new DeleteNamespaceRequest().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new DeleteNamespaceRequest().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new DeleteNamespaceRequest().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(DeleteNamespaceRequest, a, b);
  }
}

class DeleteNamespaceResponse extends Message {
  constructor(data) {
    super();
    proto3.util.initPartial(data, this);
  }
  static runtime = proto3;
  static typeName = "api.v1.DeleteNamespaceResponse";
  static fields = proto3.util.newFieldList(() => []);
  static fromBinary(bytes, options) {
    return new DeleteNamespaceResponse().fromBinary(bytes, options);
  }
  static fromJson(jsonValue, options) {
    return new DeleteNamespaceResponse().fromJson(jsonValue, options);
  }
  static fromJsonString(jsonString, options) {
    return new DeleteNamespaceResponse().fromJsonString(jsonString, options);
  }
  static equals(a, b) {
    return proto3.util.equals(DeleteNamespaceResponse, a, b);
  }
}

// js/api/v1/ipam_connect.ts
var IpamService = {
  typeName: "api.v1.IpamService",
  methods: {
    createPrefix: {
      name: "CreatePrefix",
      I: CreatePrefixRequest,
      O: CreatePrefixResponse,
      kind: MethodKind.Unary
    },
    deletePrefix: {
      name: "DeletePrefix",
      I: DeletePrefixRequest,
      O: DeletePrefixResponse,
      kind: MethodKind.Unary
    },
    getPrefix: {
      name: "GetPrefix",
      I: GetPrefixRequest,
      O: GetPrefixResponse,
      kind: MethodKind.Unary
    },
    listPrefixes: {
      name: "ListPrefixes",
      I: ListPrefixesRequest,
      O: ListPrefixesResponse,
      kind: MethodKind.Unary
    },
    prefixUsage: {
      name: "PrefixUsage",
      I: PrefixUsageRequest,
      O: PrefixUsageResponse,
      kind: MethodKind.Unary
    },
    acquireChildPrefix: {
      name: "AcquireChildPrefix",
      I: AcquireChildPrefixRequest,
      O: AcquireChildPrefixResponse,
      kind: MethodKind.Unary
    },
    releaseChildPrefix: {
      name: "ReleaseChildPrefix",
      I: ReleaseChildPrefixRequest,
      O: ReleaseChildPrefixResponse,
      kind: MethodKind.Unary
    },
    acquireIP: {
      name: "AcquireIP",
      I: AcquireIPRequest,
      O: AcquireIPResponse,
      kind: MethodKind.Unary
    },
    releaseIP: {
      name: "ReleaseIP",
      I: ReleaseIPRequest,
      O: ReleaseIPResponse,
      kind: MethodKind.Unary
    },
    dump: {
      name: "Dump",
      I: DumpRequest,
      O: DumpResponse,
      kind: MethodKind.Unary
    },
    load: {
      name: "Load",
      I: LoadRequest,
      O: LoadResponse,
      kind: MethodKind.Unary
    },
    createNamespace: {
      name: "CreateNamespace",
      I: CreateNamespaceRequest,
      O: CreateNamespaceResponse,
      kind: MethodKind.Unary
    },
    listNamespaces: {
      name: "ListNamespaces",
      I: ListNamespacesRequest,
      O: ListNamespacesResponse,
      kind: MethodKind.Unary
    },
    deleteNamespace: {
      name: "DeleteNamespace",
      I: DeleteNamespaceRequest,
      O: DeleteNamespaceResponse,
      kind: MethodKind.Unary
    }
  }
};

// www/index.ts
var refreshPrefixes = function(prefix) {
  const divEl = document.createElement("div");
  const pEl = document.createElement("p");
  const respContainerEl = containerEl.appendChild(divEl);
  respContainerEl.className = `prefix-resp-container`;
  const respTextEl = respContainerEl.appendChild(pEl);
  respTextEl.className = "resp-text";
  if (prefix !== undefined) {
    respTextEl.innerText = prefix.cidr;
  } else {
    respTextEl.innerText = "Unknown CIDR";
  }
};
async function listPrefixes() {
  const request = new ListPrefixesRequest({});
  const response = await client.listPrefixes(request);
  for (let index = 0;index < response.prefixes.length; index++) {
    const prefix = response.prefixes[index];
    console.log("prefix: ${prefix}");
    refreshPrefixes(prefix);
  }
}
async function createPrefix() {
  const cidr = inputEl?.value ?? "";
  inputEl.value = "";
  const request = new CreatePrefixRequest({
    cidr
  });
  const response = await client.createPrefix(request);
  listPrefixes();
  console.log("prefix created:" + response.prefix?.cidr);
}
var client = createPromiseClient(IpamService, createConnectTransport({
  baseUrl: "http://localhost:9090"
}));
var containerEl = document.getElementById("conversation-container");
var inputEl = document.getElementById("user-input");
document.getElementById("user-input")?.addEventListener("keyup", (event) => {
  event.preventDefault();
  if (event.key === "Enter") {
    createPrefix();
  }
});
document.getElementById("send-button")?.addEventListener("click", (event) => {
  event.preventDefault();
  createPrefix();
});
