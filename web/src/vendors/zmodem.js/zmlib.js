"use strict";

var Zmodem = module.exports;

const
    ZDLE = 0x18,
    XON = 0x11,
    XOFF = 0x13,
    XON_HIGH = 0x80 | XON,
    XOFF_HIGH = 0x80 | XOFF,
    CAN = 0x18     //NB: same character as ZDLE
;

/**
 * Tools and constants that are useful for ZMODEM.
 *
 * @exports ZMLIB
 */
Zmodem.ZMLIB = {

    /**
     * @property {number} The ZDLE constant, which ZMODEM uses for escaping
     */
    ZDLE: ZDLE,

    /**
     * @property {number} XON - ASCII XON
     */
    XON: XON,

    /**
     * @property {number} XOFF - ASCII XOFF
     */
    XOFF: XOFF,

    /**
     * @property {number[]} ABORT_SEQUENCE - ZMODEM’s abort sequence
     */
    ABORT_SEQUENCE: [ CAN, CAN, CAN, CAN, CAN ],

    /**
     * Remove octet values from the given array that ZMODEM always ignores.
     * This will mutate the given array.
     *
     * @param {number[]} octets - The octet values to transform.
     *      Each array member should be an 8-bit unsigned integer (0-255).
     *      This object is mutated in the function.
     *
     * @returns {number[]} The passed-in array. This is the same object that is
     *      passed in.
     */
    strip_ignored_bytes: function strip_ignored_bytes(octets) {
        for (var o=octets.length-1; o>=0; o--) {
            switch (octets[o]) {
                case XON:
                case XON_HIGH:
                case XOFF:
                case XOFF_HIGH:
                    octets.splice(o, 1);
                    continue;
            }
        }

        return octets;
    },

    /**
     * Like Array.prototype.indexOf, but searches for a subarray
     * rather than just a particular value.
     *
     * @param {Array} haystack - The array to search, i.e., the bigger.
     *
     * @param {Array} needle - The array whose values to find,
     *      i.e., the smaller.
     *
     * @returns {number} The position in “haystack” where “needle”
     *      first appears—or, -1 if “needle” doesn’t appear anywhere
     *      in “haystack”.
     */
    find_subarray: function find_subarray(haystack, needle) {
        var h=0, n;

        var start = Date.now();

        HAYSTACK:
        while (h !== -1) {
            h = haystack.indexOf( needle[0], h );
            if (h === -1) break HAYSTACK;

            for (n=1; n<needle.length; n++) {
                if (haystack[h + n] !== needle[n]) {
                    h++;
                    continue HAYSTACK;
                }
            }

            return h;
        }

        return -1;
    },
};
