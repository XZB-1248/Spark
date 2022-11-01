"use strict";

var Zmodem = module.exports;

Object.assign(
    Zmodem,
    require("./zcrc"),
    require("./zdle"),
    require("./zmlib"),
    require("./zerror")
);

const
    ZCRCE = 0x68,    // 'h', 104, frame ends, header packet follows
    ZCRCG = 0x69,    // 'i', 105, frame continues nonstop
    ZCRCQ = 0x6a,    // 'j', 106, frame continues, ZACK expected
    ZCRCW = 0x6b     // 'k', 107, frame ends, ZACK expected
;

var SUBPACKET_BUILDER;

/** Class that represents a ZMODEM data subpacket. */
Zmodem.Subpacket = class ZmodemSubpacket {

    /**
     * Build a Subpacket subclass given a payload and frame end string.
     *
     * @param {Array} octets - The octet values to parse.
     *      Each array member should be an 8-bit unsigned integer (0-255).
     *
     * @param {string} frameend - One of:
     * - `no_end_no_ack`
     * - `end_no_ack`
     * - `no_end_ack` (unused currently)
     * - `end_ack`
     *
     * @returns {Subpacket} An instance of the appropriate Subpacket subclass.
     */
    static build(octets, frameend) {

        //TODO: make this better
        var Ctr = SUBPACKET_BUILDER[frameend];
        if (!Ctr) {
            throw("No subpacket type “" + frameend + "” is defined! Try one of: " + Object.keys(SUBPACKET_BUILDER).join(", "));
        }

        return new Ctr(octets);
    }

    /**
     * Return the octet values array that represents the object
     * encoded with a 16-bit CRC.
     *
     * @param {ZDLE} zencoder - A ZDLE instance to use for ZDLE encoding.
     *
     * @returns {number[]} An array of octet values suitable for sending
     *      as binary data.
     */
    encode16(zencoder) {
        return this._encode( zencoder, Zmodem.CRC.crc16 );
    }

    /**
     * Return the octet values array that represents the object
     * encoded with a 32-bit CRC.
     *
     * @param {ZDLE} zencoder - A ZDLE instance to use for ZDLE encoding.
     *
     * @returns {number[]} An array of octet values suitable for sending
     *      as binary data.
     */
    encode32(zencoder) {
        return this._encode( zencoder, Zmodem.CRC.crc32 );
    }

    /**
     * Return the subpacket payload’s octet values.
     *
     * NOTE: For speed, this returns the actual data in the subpacket;
     * if you mutate this return value, you alter the Subpacket object
     * internals. This is OK if you won’t need the Subpacket anymore, but
     * just be careful.
     *
     * @returns {number[]} The subpacket’s payload, represented as an
     * array of octet values. **DO NOT ALTER THIS ARRAY** unless you
     * no longer need the Subpacket.
     */
    get_payload() { return this._payload }

    /**
     * Parse out a Subpacket object from a given array of octet values,
     * assuming a 16-bit CRC.
     *
     * An exception is thrown if the given bytes are definitively invalid
     * as subpacket values with 16-bit CRC.
     *
     * @param {number[]} octets - The octet values to parse.
     *      Each array member should be an 8-bit unsigned integer (0-255).
     *      This object is mutated in the function.
     *
     * @returns {Subpacket|undefined} An instance of the appropriate Subpacket
     *      subclass, or undefined if not enough octet values are given
     *      to determine whether there is a valid subpacket here or not.
     */
    static parse16(octets) {
        return ZmodemSubpacket._parse(octets, 2);
    }

    //parse32 test:
    //[102, 105, 108, 101, 110, 97, 109, 101, 119, 105, 116, 104, 115, 112, 97, 99, 101, 115, 0, 49, 55, 49, 51, 49, 52, 50, 52, 51, 50, 49, 55, 50, 49, 48, 48, 54, 52, 52, 48, 49, 49, 55, 0, 43, 8, 63, 115, 23, 17]

    /**
     * Same as parse16(), but assuming a 32-bit CRC.
     *
     * @param {number[]} octets - The octet values to parse.
     *      Each array member should be an 8-bit unsigned integer (0-255).
     *      This object is mutated in the function.
     *
     * @returns {Subpacket|undefined} An instance of the appropriate Subpacket
     *      subclass, or undefined if not enough octet values are given
     *      to determine whether there is a valid subpacket here or not.
     */
    static parse32(octets) {
        return ZmodemSubpacket._parse(octets, 4);
    }

    /**
     * Not used directly.
     */
    constructor(payload) {
        this._payload = payload;
    }

    _encode(zencoder, crc_func) {
        return zencoder.encode( this._payload.slice(0) ).concat(
            [ Zmodem.ZMLIB.ZDLE, this._frameend_num ],
            zencoder.encode( crc_func( this._payload.concat(this._frameend_num) ) )
        );
    }

    //Because of ZDLE encoding, we’ll never see any of the frame-end octets
    //in a stream except as the ends of data payloads.
    static _parse(bytes_arr, crc_len) {

        var end_at;
        var creator;

        //These have to be written in decimal since they’re lookup keys.
        var _frame_ends_lookup = {
            104: ZEndNoAckSubpacket,
            105: ZNoEndNoAckSubpacket,
            106: ZNoEndAckSubpacket,
            107: ZEndAckSubpacket,
        };

        var zdle_at = 0;
        while (zdle_at < bytes_arr.length) {
            zdle_at = bytes_arr.indexOf( Zmodem.ZMLIB.ZDLE, zdle_at );
            if (zdle_at === -1) return;

            var after_zdle = bytes_arr[ zdle_at + 1 ];
            creator = _frame_ends_lookup[ after_zdle ];
            if (creator) {
                end_at = zdle_at + 1;
                break;
            }

            zdle_at++;
        }

        if (!creator) return;

        var frameend_num = bytes_arr[end_at];

        //sanity check
        if (bytes_arr[end_at - 1] !== Zmodem.ZMLIB.ZDLE) {
            throw( "Byte before frame end should be ZDLE, not " + bytes_arr[end_at - 1] );
        }

        var zdle_encoded_payload = bytes_arr.splice( 0, end_at - 1 );

        var got_crc = Zmodem.ZDLE.splice( bytes_arr, 2, crc_len );
        if (!got_crc) {
            //got payload but no CRC yet .. should be rare!

            //We have to put the ZDLE-encoded payload back before returning.
            bytes_arr.unshift.apply(bytes_arr, zdle_encoded_payload);

            return;
        }

        var payload = Zmodem.ZDLE.decode(zdle_encoded_payload);

        //We really shouldn’t need to do this, but just for good measure.
        //I suppose it’s conceivable this may run over UDP or something?
        Zmodem.CRC[ (crc_len === 2) ? "verify16" : "verify32" ](
            payload.concat( [frameend_num] ),
            got_crc
        );

        return new creator(payload, got_crc);
    }
}

class ZEndSubpacketBase extends Zmodem.Subpacket {
    frame_end() { return true }
}
class ZNoEndSubpacketBase extends Zmodem.Subpacket {
    frame_end() { return false }
}

//Used for end-of-file.
class ZEndNoAckSubpacket extends ZEndSubpacketBase {
    ack_expected() { return false }
}
ZEndNoAckSubpacket.prototype._frameend_num = ZCRCE;

//Used for ZFILE and ZSINIT payloads.
class ZEndAckSubpacket extends ZEndSubpacketBase {
    ack_expected() { return true }
}
ZEndAckSubpacket.prototype._frameend_num = ZCRCW;

//Used for ZDATA, prior to end-of-file.
class ZNoEndNoAckSubpacket extends ZNoEndSubpacketBase {
    ack_expected() { return false }
}
ZNoEndNoAckSubpacket.prototype._frameend_num = ZCRCG;

//only used if receiver can full-duplex
class ZNoEndAckSubpacket extends ZNoEndSubpacketBase {
    ack_expected() { return true }
}
ZNoEndAckSubpacket.prototype._frameend_num = ZCRCQ;

SUBPACKET_BUILDER = {
    end_no_ack: ZEndNoAckSubpacket,
    end_ack: ZEndAckSubpacket,
    no_end_no_ack: ZNoEndNoAckSubpacket,
    no_end_ack: ZNoEndAckSubpacket,
};
