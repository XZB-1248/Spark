"use strict";

var Zmodem = module.exports;

function _crc_message(got, expected) {
    this.got = got.slice(0);
    this.expected = expected.slice(0);
    return "CRC check failed! (got: " + got.join() + "; expected: " + expected.join() + ")";
}

function _pass(val) { return val }

const TYPE_MESSAGE = {
    aborted: "Session aborted",
    peer_aborted: "Peer aborted session",
    already_aborted: "Session already aborted",
    crc: _crc_message,
    validation: _pass,
};

function _generate_message(type) {
    const msg = TYPE_MESSAGE[type];
    switch (typeof msg) {
        case "string":
            return msg;
        case "function":
            var args_after_type = [].slice.call(arguments).slice(1);
            return msg.apply(this, args_after_type);
    }

    return null;
}

Zmodem.Error = class ZmodemError extends Error {
    constructor(msg_or_type) {
        super();

        var generated = _generate_message.apply(this, arguments);
        if (generated) {
            this.type = msg_or_type;
            this.message = generated;
        }
        else {
            this.message = msg_or_type;
        }
    }
};
