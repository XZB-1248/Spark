class _my_TextEncoder {
    encode(text) {
        text = unescape(encodeURIComponent(text));

        var bytes = new Array( text.length );

        for (var b = 0; b < text.length; b++) {
            bytes[b] = text.charCodeAt(b);
        }

        return new Uint8Array(bytes);
    }
}

class _my_TextDecoder {
    decode(bytes) {
        return decodeURIComponent( escape( String.fromCharCode.apply(String, bytes) ) );
    }
}

var Zmodem = module.exports;

/**
 * A limited-use compatibility shim for TextEncoder and TextDecoder.
 * Useful because both Edge and node.js still lack support for these
 * as of October 2017.
 *
 * @exports Text
 */
Zmodem.Text = {
    Encoder: (typeof TextEncoder !== "undefined") ? TextEncoder : _my_TextEncoder,
    Decoder: (typeof TextDecoder !== "undefined") ? TextDecoder : _my_TextDecoder,
};
