"use strict";

const CRC32_MOD = require('crc-32');

var Zmodem = module.exports;

Object.assign(
    Zmodem,
    require("./zerror"),
    require("./encode")
);

//----------------------------------------------------------------------
// BEGIN adapted from crc-js by Johannes Rudolph

var _crctab;

const
    crc_width = 16,
    crc_polynomial = 0x1021,
    crc_castmask = 0xffff,
    crc_msbmask = 1 << (crc_width - 1)
;

function _compute_crctab() {
    _crctab = new Array(256);

    var divident_shift = crc_width - 8;

    for (var divident = 0; divident < 256; divident++) {
        var currByte = (divident << divident_shift) & crc_castmask;

        for (var bit = 0; bit < 8; bit++) {

            if ((currByte & crc_msbmask) !== 0) {
                currByte <<= 1;
                currByte ^= crc_polynomial;
            }
            else {
                currByte <<= 1;
            }
        }

        _crctab[divident] = (currByte & crc_castmask);
    }
}

// END adapted from crc-js by Johannes Rudolph
//----------------------------------------------------------------------

function _updcrc(cp, crc) {
    if (!_crctab) _compute_crctab();

    return(
        _crctab[((crc >> 8) & 255)]
        ^ ((255 & crc) << 8)
        ^ cp
    );
}

function __verify(expect, got) {
    var err;

    if ( expect.join() !== got.join() ) {
        throw new Zmodem.Error("crc", got, expect);
    }
}

//TODO: use external implementation(s)
Zmodem.CRC = {

    //https://www.lammertbies.nl/comm/info/crc-calculation.html
    //CRC-CCITT (XModem)

    /**
     * Deduce a given set of octet values’ CRC16, as per the CRC16
     * variant that ZMODEM uses (CRC-CCITT/XModem).
     *
     * @param {Array} octets - The array of octet values.
     *      Each array member should be an 8-bit unsigned integer (0-255).
     *
     * @returns {Array} crc - The CRC, expressed as an array of octet values.
     */
    crc16: function crc16(octet_nums) {
        var crc = octet_nums[0];
        for (var b=1; b<octet_nums.length; b++) {
            crc = _updcrc( octet_nums[b], crc );
        }

        crc = _updcrc( 0, _updcrc(0, crc) );

        //a big-endian 2-byte sequence
        return Zmodem.ENCODELIB.pack_u16_be(crc);
    },

    /**
     * Deduce a given set of octet values’ CRC32.
     *
     * @param {Array} octets - The array of octet values.
     *      Each array member should be an 8-bit unsigned integer (0-255).
     *
     * @returns {Array} crc - The CRC, expressed as an array of octet values.
     */
    crc32: function crc32(octet_nums) {
        return Zmodem.ENCODELIB.pack_u32_le(
            CRC32_MOD.buf(octet_nums) >>> 0     //bit-shift to get unsigned
        );
    },

    /**
     * Verify a given set of octet values’ CRC16.
     * An exception is thrown on failure.
     *
     * @param {Array} bytes_arr - The array of octet values.
     *      Each array member should be an 8-bit unsigned integer (0-255).
     *
     * @param {Array} crc - The CRC to check against, expressed as
     *      an array of octet values.
     */
    verify16: function verify16(bytes_arr, got) {
        return __verify( this.crc16(bytes_arr), got );
    },

    /**
     * Verify a given set of octet values’ CRC32.
     * An exception is thrown on failure.
     *
     * @param {Array} bytes_arr - The array of octet values.
     *      Each array member should be an 8-bit unsigned integer (0-255).
     *
     * @param {Array} crc - The CRC to check against, expressed as
     *      an array of octet values.
     */
    verify32: function verify32(bytes_arr, crc) {
        try {
            __verify( this.crc32(bytes_arr), crc );
        }
        catch(err) {
            err.input = bytes_arr.slice(0);
            throw err;
        }
    },
};
