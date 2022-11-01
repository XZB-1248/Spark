"use strict";

var Zmodem = module.exports;

/**
 * This is where the protocol-level logic lives: the interaction of ZMODEM
 * headers and subpackets. The logic here is not unlikely to need tweaking
 * as little edge cases crop up.
 */

Zmodem.DEBUG = false;

Object.assign(
    Zmodem,
    require("./encode"),
    require("./text"),
    require("./zdle"),
    require("./zmlib"),
    require("./zheader"),
    require("./zsubpacket"),
    require("./zvalidation"),
    require("./zerror")
);

const
    //pertinent to this module
    KEEPALIVE_INTERVAL = 5000,

    //We ourselves don’t need ESCCTL, so we don’t send it;
    //however, we always expect to receive it in ZRINIT.
    //See _ensure_receiver_escapes_ctrl_chars() for more details.
    ZRINIT_FLAGS = [
        "CANFDX",   //full duplex
        "CANOVIO",  //overlap I/O

        //lsz has a buffer overflow bug that shows itself when:
        //
        //  - 16-bit CRC is used, and
        //  - lsz receives the abort sequence while sending a file
        //
        //To avoid this, we just tell lsz to use 32-bit CRC
        //even though there is otherwise no reason. This ensures that
        //unfixed lsz versions will avoid the buffer overflow.
        "CANFC32",
    ],

    //We do this because some WebSocket shell servers
    //(e.g., xterm.js’s demo server) enable the IEXTEN termios flag,
    //which bars 0x0f and 0x16 from reaching the shell process,
    //which results in transmission errors.
    FORCE_ESCAPE_CTRL_CHARS = true,

    DEFAULT_RECEIVE_INPUT_MODE = "spool_uint8array",

    //pertinent to ZMODEM
    MAX_CHUNK_LENGTH = 8192,    //1 KiB officially, but lrzsz allows 8192
    BS = 0x8,
    OVER_AND_OUT = [ 79, 79 ],
    ABORT_SEQUENCE = Zmodem.ZMLIB.ABORT_SEQUENCE
;

/**
 * A base class for objects that have events.
 *
 * @private
 */
class _Eventer {

    /**
     * Not called directly.
     */
    constructor() {
        this._on_evt = {};
        this._evt_once_index = {};
    }

    _Add_event(evt_name) {
        this._on_evt[evt_name] = [];
        this._evt_once_index[evt_name] = [];
    }

    _get_evt_queue(evt_name) {
        if (!this._on_evt[evt_name]) {
            throw( "Bad event: " + evt_name );
        }

        return this._on_evt[evt_name];
    }

    /**
     * Register a callback for a given event.
     *
     * @param {string} evt_name - The name of the event.
     *
     * @param {Function} todo - The function to execute when the event happens.
     */
    on(evt_name, todo) {
        var queue = this._get_evt_queue(evt_name);

        queue.push(todo);

        return this;
    }

    /**
     * Unregister a callback for a given event.
     *
     * @param {string} evt_name - The name of the event.
     *
     * @param {Function} [todo] - The function to execute when the event
     *  happens. If not given, the last event registered for the event
     *  is unregistered.
     */
    off(evt_name, todo) {
        var queue = this._get_evt_queue(evt_name);

        if (todo) {
            var at = queue.indexOf(todo);
            if (at === -1) {
                throw("“" + todo + "” is not in the “" + evt_name + "” queue.");
            }
            queue.splice(at, 1);
        }
        else {
            queue.pop();
        }

        return this;
    }

    _Happen(evt_name /*, arg0, arg1, .. */) {
        var queue = this._get_evt_queue(evt_name);   //might as well validate

        //console.info("EVENT", this, arguments);

        var args = Array.apply(null, arguments);
        args.shift();

        var sess = this;

        queue.forEach( function(cb) { cb.apply(sess, args) } );

        return queue.length;
    }
}

/**
 * The Session classes handle the protocol-level logic.
 * These shield the user from dealing with headers and subpackets.
 * This is a base class with functionality common to both Receive
 * and Send subclasses.
 *
 * @extends _Eventer
*/
Zmodem.Session = class ZmodemSession extends _Eventer {

    /**
     * Parse out a hex header from the given array.
     * If there’s a ZRQINIT or ZRINIT at the beginning,
     * we’ll return it. If the input isn’t a header,
     * for whatever reason, we return undefined.
     *
     * @param {number[]} octets - The bytes to parse.
     *
     * @return {Session|undefined} A Session object if the beginning
     *      of a session was parsable in “octets”; otherwise undefined.
     */
    static parse( octets ) {

        //Will need to trap errors.
        var hdr;
        try {
            hdr = Zmodem.Header.parse_hex(octets);
        }
        catch(e) {     //Don’t report since we aren’t in session

            //debug
            //console.warn("No hex header: ", e);

            return;
        }

        if (!hdr) return;

        switch (hdr.NAME) {
            case "ZRQINIT":
                //throw if ZCOMMAND
                return new Zmodem.Session.Receive();
            case "ZRINIT":
                return new Zmodem.Session.Send(hdr);
        }

        //console.warn("Invalid first Zmodem header", hdr);
    }

    /**
     * Sets the sender function that a Session object will use.
     *
     * @param {Function} sender_func - The function to call.
     *      It will receive an Array with the relevant octets.
     *
     * @return {Session} The session object (for chaining).
     */
    set_sender(sender_func) {
        this._sender = sender_func;
        return this;
    }

    /**
     * Whether the current Session has ended.
     *
     * @returns {boolean} The ended state.
     */
    has_ended() { return this._has_ended() }

    /**
     * Consumes an array of octets as ZMODEM session input.
     *
     * @param {number[]} octets - The input octets.
     */
    consume(octets) {
        this._before_consume(octets);

        if (this._aborted) throw new Zmodem.Error('already_aborted');

        if (!octets.length) return;

        this._strip_and_enqueue_input(octets);

        if (!this._check_for_abort_sequence(octets)) {
            this._consume_first();
        }

        return;
    }

    /**
     * Whether the current Session has been `abort()`ed.
     *
     * @returns {boolean} The aborted state.
     */
    aborted() { return !!this._aborted }

    /**
     * Not called directly.
     */
    constructor() {
        super();
        //if (!sender_func) throw "Need sender!";

        //this._first_header = first_header;
        //this._sender = sender_func;
        this._config = {};

        //this._input = new ZInput();

        this._input_buffer = [];

        //This is mostly for debugging.
        this._Add_event("receive");
        this._Add_event("garbage");
        this._Add_event("session_end");
    }

    /**
     * Returns the Session object’s role.
     *
     * @returns {string} One of:
     * - `receive`
     * - `send`
     */
    get_role() { return this.type }

    _trim_leading_garbage_until_header() {
        var garbage = Zmodem.Header.trim_leading_garbage(this._input_buffer);

        if (garbage.length) {
            if (this._Happen("garbage", garbage) === 0) {
                console.debug(
                    "Garbage: ",
                    String.fromCharCode.apply(String, garbage),
                    garbage
                );
            }
        }
    }

    _parse_and_consume_header() {
        this._trim_leading_garbage_until_header();

        var new_header_and_crc = Zmodem.Header.parse(this._input_buffer);
        if (!new_header_and_crc) return;

        if (Zmodem.DEBUG) {
            this._log_header( "RECEIVED HEADER", new_header_and_crc[0] );
        }

        this._consume_header(new_header_and_crc[0]);

        this._last_header_name = new_header_and_crc[0].NAME;
        this._last_header_crc = new_header_and_crc[1];

        return new_header_and_crc[0];
    }

    _log_header(label, header) {
        console.debug(this.type, label, header.NAME, header._bytes4.join());
    }

    _consume_header(new_header) {
        this._on_receive(new_header);

        var handler = this._next_header_handler && this._next_header_handler[ new_header.NAME ];
        if (!handler) {
            console.error("Unhandled header!", new_header, this._next_header_handler);
            throw new Zmodem.Error( "Unhandled header: " + new_header.NAME );
        }

        this._next_header_handler = null;

        handler.call(this, new_header);
    }

    //TODO: strip out the abort sequence
    _check_for_abort_sequence() {
        var abort_at = Zmodem.ZMLIB.find_subarray( this._input_buffer, ABORT_SEQUENCE );

        if (abort_at !== -1) {

            //TODO: expose this to caller
            this._input_buffer.splice( 0, abort_at + ABORT_SEQUENCE.length );

            this._aborted = true;

            //TODO compare response here to lrzsz.
            this._on_session_end();

            //We shouldn’t ever expect to receive an abort. Even if we
            //have sent an abort ourselves, the Sentry should have stopped
            //directing input to this Session object.
            //if (this._expect_abort) {
            //    return true;
            //}

            throw new Zmodem.Error("peer_aborted");
        }
    }

    _send_header(name /*, args */) {
        if (!this._sender) throw "Need sender!";

        var args = Array.apply( null, arguments );

        var bytes_hdr = this._create_header_bytes(args);

        if (Zmodem.DEBUG) {
            this._log_header( "SENDING HEADER", bytes_hdr[1] );
        }

        this._sender(bytes_hdr[0]);

        this._last_sent_header = bytes_hdr[1];
    }

    _create_header_bytes(name_and_args) {

        var hdr = Zmodem.Header.build.apply( Zmodem.Header, name_and_args );

        var formatter = this._get_header_formatter(name_and_args[0]);

        return [
            hdr[formatter](this._zencoder),
            hdr
        ];
    }

    _strip_and_enqueue_input(input) {
        Zmodem.ZMLIB.strip_ignored_bytes(input);

        //It’s possible that “input” is empty at this point.
        //It doesn’t seem to hurt anything to keep processing, though.

        this._input_buffer.push.apply( this._input_buffer, input );
    }

    /**
     * **STOP!** You probably want to `skip()` an Offer rather than
     * `abort()`. See below.
     *
     * Abort the current session by sending the ZMODEM abort sequence.
     * This function will cause the Session object to refuse to send
     * any further data.
     *
     * Zmodem.Sentry is configured to send all output to the terminal
     * after a session’s `abort()`. That could result in lots of
     * ZMODEM garble being sent to the JavaScript terminal, which you
     * probably don’t want.
     *
     * `skip()` on an Offer is better because Session will continue to
     * discard data until we reach either another file or the
     * sender-initiated end of the ZMODEM session. So no ZMODEM garble,
     * and the session will end successfully.
     *
     * The behavior of `abort()` is subject to change since it’s not
     * very useful as currently implemented.
     */
    abort() {

        //this._expect_abort = true;

        //From Forsberg:
        //
        //The Cancel sequence consists of eight CAN characters
        //and ten backspace characters. ZMODEM only requires five
        //Cancel characters; the other three are "insurance".
        //The trailing backspace characters attempt to erase
        //the effects of the CAN characters if they are
        //received by a command interpreter.
        //
        //FG: Since we assume our connection is reliable, there’s
        //no reason to send more than 5 CANs.
        this._sender(
            ABORT_SEQUENCE.concat([ BS, BS, BS, BS, BS ])
        );

        this._aborted = true;
        this._sender = function() {
            throw new Zmodem.Error('already_aborted');
        };

        this._on_session_end();

        return;
    }

    //----------------------------------------------------------------------
    _on_session_end() {
        this._Happen("session_end");
    }

    _on_receive(hdr_or_pkt) {
        this._Happen("receive", hdr_or_pkt);
    }

    _before_consume() {}
}

function _trim_OO(array) {
    if (0 === Zmodem.ZMLIB.find_subarray(array, OVER_AND_OUT)) {
        array.splice(0, OVER_AND_OUT.length);
    }

    //TODO: This assumes OVER_AND_OUT is 2 bytes long. No biggie, but.
    else if ( array[0] === OVER_AND_OUT[ OVER_AND_OUT.length - 1 ] ) {
        array.splice(0, 1);
    }

    return array;
}

/** A class for ZMODEM receive sessions.
 *
 * @extends Session
 */
Zmodem.Session.Receive = class ZmodemReceiveSession extends Zmodem.Session {
    //We only get 1 file at a time, so on each consume() either
    //continue state for the current file or start a new one.

    /**
     * Not called directly.
     */
    constructor() {
        super();

        this._Add_event("offer");
        this._Add_event("data_in");
        this._Add_event("file_end");
    }

    /**
     * Consume input bytes from the sender.
     *
     * @private
     * @param {number[]} octets - The bytes to consume.
     */
    _before_consume(octets) {
        if (this._bytes_after_OO) {
            throw "PROTOCOL: Session is completed!";
        }

        //Put this here so that our logic later on has access to the
        //input string and can populate _bytes_after_OO when the
        //session ends.
        this._bytes_being_consumed = octets;
    }

    /**
     * Return any bytes that have been `consume()`d but
     * came after the end of the ZMODEM session.
     *
     * @returns {number[]} The trailing bytes.
     */
    get_trailing_bytes() {
        if (this._aborted) return [];

        if (!this._bytes_after_OO) {
            throw "PROTOCOL: Session is not completed!";
        }

        return this._bytes_after_OO.slice(0);
    }

    _has_ended() { return this.aborted() || !!this._bytes_after_OO }

    //Receiver always sends hex headers.
    _get_header_formatter() { return "to_hex" }

    _parse_and_consume_subpacket() {
        var parse_func;
        if (this._last_header_crc === 16) {
            parse_func = "parse16";
        }
        else {
            parse_func = "parse32";
        }

        var subpacket = Zmodem.Subpacket[parse_func](this._input_buffer);

        if (subpacket) {
            if (Zmodem.DEBUG) {
                console.debug(this.type, "RECEIVED SUBPACKET", subpacket);
            }

            this._consume_data(subpacket);

            //What state are we in if the subpacket indicates frame end
            //but we haven’t gotten ZEOF yet? Can anything other than ZEOF
            //follow after a ZDATA?
            if (subpacket.frame_end()) {
                this._next_subpacket_handler = null;
            }
        }

        return subpacket;
    }

    _consume_first() {
        if (this._got_ZFIN) {
            if (this._input_buffer.length < 2) return;

            // some lrzsz don't send OO after ZFIN
            // whether there's OO or not, we're done
            this._bytes_after_OO = _trim_OO(this._bytes_being_consumed.slice(0));
            this._on_session_end();
            return;

            //if it’s OO, then set this._bytes_after_OO
            // if (Zmodem.ZMLIB.find_subarray(this._input_buffer, OVER_AND_OUT) === 0) {

                //This doubles as an indication that the session has ended.
                //We need to set this right away so that handlers like
                //"session_end" will have access to it.
                // this._bytes_after_OO = _trim_OO(this._bytes_being_consumed.slice(0));
                // this._on_session_end();
                //
                // return;
            // }
            // else {
            //     throw( "PROTOCOL: Only thing after ZFIN should be “OO” (79,79), not: " + this._input_buffer.join() );
            // }
        }

        var parsed;
        do {
            if (this._next_subpacket_handler) {
                parsed = this._parse_and_consume_subpacket();
            }
            else {
                parsed = this._parse_and_consume_header();
            }
        } while (parsed && this._input_buffer.length);
    }

    _consume_data(subpacket) {
        this._on_receive(subpacket);

        if (!this._next_subpacket_handler) {
            throw( "PROTOCOL: Received unexpected data packet after " + this._last_header_name + " header: " + subpacket.get_payload().join() );
        }

        this._next_subpacket_handler.call(this, subpacket);
    }

    _octets_to_string(octets) {
        if (!this._textdecoder) {
            this._textdecoder = new Zmodem.Text.Decoder();
        }

        return this._textdecoder.decode( new Uint8Array(octets) );
    }

    _consume_ZFILE_data(hdr, subpacket) {
        if (this._file_info) {
            throw "PROTOCOL: second ZFILE data subpacket received";
        }

        var packet_payload = subpacket.get_payload();
        var nul_at = packet_payload.indexOf(0);

        //
        var fname = this._octets_to_string( packet_payload.slice(0, nul_at) );
        var the_rest = this._octets_to_string( packet_payload.slice( 1 + nul_at ) ).split(" ");

        var mtime = the_rest[1] && parseInt( the_rest[1], 8 ) || undefined;
        if (mtime) {
            mtime = new Date(mtime * 1000);
        }

        this._file_info = {
            name: fname,
            size: the_rest[0] ? parseInt( the_rest[0], 10 ) : null,
            mtime: mtime || null,
            mode: the_rest[2] && parseInt( the_rest[2], 8 ) || null,
            serial: the_rest[3] && parseInt( the_rest[3], 10 ) || null,

            files_remaining: the_rest[4] ? parseInt( the_rest[4], 10 ) : null,
            bytes_remaining: the_rest[5] ? parseInt( the_rest[5], 10 ) : null,
        };

        //console.log("ZFILE", hdr);

        var xfer = new Offer(
            hdr.get_options(),
            this._file_info,
            this._accept.bind(this),
            this._skip.bind(this)
        );
        this._current_transfer = xfer;

        //this._Happen("offer", xfer);
    }

    _consume_ZDATA_data(subpacket) {
        if (!this._accepted_offer) {
            throw "PROTOCOL: Received data without accepting!";
        }

        //TODO: Probably should include some sort of preventive against
        //infinite loop here: if the peer hasn’t sent us what we want after,
        //say, 10 ZRPOS headers then we should send ZABORT and just end.
        if (!this._offset_ok) {
            console.warn("offset not ok!");
            _send_ZRPOS();
            return;
        }

        this._file_offset += subpacket.get_payload().length;
        this._on_data_in(subpacket);

        /*
        console.warn("received error from data_in callback; retrying", e);
        throw "unimplemented";
        */

        if (subpacket.ack_expected() && !subpacket.frame_end()) {
            this._send_header( "ZACK", Zmodem.ENCODELIB.pack_u32_le(this._file_offset) );
        }
    }

    _make_promise_for_between_files() {
        var sess = this;

        return new Promise( function(res) {
            var between_files_handler = {
                ZFILE: function(hdr) {
                    this._next_subpacket_handler = function(subpacket) {
                        this._next_subpacket_handler = null;
                        this._consume_ZFILE_data(hdr, subpacket);
                        this._Happen("offer", this._current_transfer);
                        res(this._current_transfer);
                    };
                },

                //We use this as a keep-alive. Maybe other
                //implementations do, too?
                ZSINIT: function(hdr) {
                    //The content of this header doesn’t affect us
                    //since all it does is tell us details of how
                    //the sender will ZDLE-encode binary data. Our
                    //ZDLE parser doesn’t need to know in advance.

                    sess._next_subpacket_handler = function(spkt) {
                        sess._next_subpacket_handler = null;
                        sess._consume_ZSINIT_data(spkt);
                        sess._send_header('ZACK');
                        sess._next_header_handler = between_files_handler;
                    };
                },

                ZFIN: function() {
                    this._consume_ZFIN();
                    res();
                },
            };

            sess._next_header_handler = between_files_handler;
        } );
    }

    _consume_ZSINIT_data(spkt) {

        //TODO: Should this be used when we signal a cancellation?
        this._attn = spkt.get_payload();
    }

    /**
     * Start the ZMODEM session by signaling to the sender that
     * we are ready for the first file offer.
     *
     * @returns {Promise} A promise that resolves with an Offer object
     * or, if the sender closes the session immediately without offering
     * anything, nothing.
     */
    start() {
        if (this._started) throw "Already started!";
        this._started = true;

        var ret = this._make_promise_for_between_files();

        this._send_ZRINIT();

        return ret;
    }

    //Returns a promise that’s fulfilled when the file
    //transfer is done.
    //
    //  That ZEOF promise return is another promise that’s
    //  fulfilled when we get either ZFIN or another ZFILE.
    _accept(offset) {
        this._accepted_offer = true;
        this._file_offset = offset || 0;

        var sess = this;

        var ret = new Promise( function(resolve_accept) {
            var last_ZDATA;

            sess._next_header_handler = {
                ZDATA: function on_ZDATA(hdr) {
                    this._consume_ZDATA(hdr);

                    this._next_subpacket_handler = this._consume_ZDATA_data;

                    this._next_header_handler = {
                        ZEOF: function on_ZEOF(hdr) {

                            // Do this first to verify the ZEOF.
                            // This also fires the “file_end” event.
                            this._consume_ZEOF(hdr);

                            this._next_subpacket_handler = null;

                            // We don’t care about this promise.
                            // Prior to v0.1.8 we did because we called
                            // resolve_accept() at the resolution of this
                            // promise, but that was a bad idea and was
                            // never documented, so 0.1.8 changed it.
                            this._make_promise_for_between_files();

                            resolve_accept();

                            this._send_ZRINIT();
                        },
                    };
                },
            };
        } );

        this._send_ZRPOS();

        return ret;
    }

    _skip() {
        var ret = this._make_promise_for_between_files();

        if (this._accepted_offer) {
            // There’s a race condition where we might attempt to
            // skip() an in-progress transfer near its end but actually
            // the skip() will fire after the transfer is complete.
            // While there might be ways to prevent this, they likely
            // would require extra work on the part of implementations.
            //
            // It seems far simpler just to make this function a no-op
            // in these cases.
            if (!this._current_transfer) return;

            //For cancel of an in-progress transfer from lsz,
            //it’s necessary to avoid this buffer overflow bug:
            //
            //  https://github.com/gooselinux/lrzsz/blob/master/lrzsz-0.12.20.patch
            //
            //… which we do by asking for CRC32 from lsz.

            //We might or might not have consumed ZDATA.
            //The sender also might or might not send a ZEOF before it
            //parses the ZSKIP. Thus, we want to ignore the following:
            //
            //  - ZDATA
            //  - ZDATA then ZEOF
            //  - ZEOF
            //
            //… and just look for the next between-file header.

            var bound_make_promise_for_between_files = function() {

                //Once this happens we fail on any received data packet.
                //So it needs not to happen until we’ve received a header.
                this._accepted_offer = false;
                this._next_subpacket_handler = null;

                this._make_promise_for_between_files();
            }.bind(this);

            Object.assign(
                this._next_header_handler,
                {
                    ZEOF: bound_make_promise_for_between_files,
                    ZDATA: function() {
                        bound_make_promise_for_between_files();
                        this._next_header_handler.ZEOF = bound_make_promise_for_between_files;
                    }.bind(this),
                }
            );
        }

        //this._accepted_offer = false;

        this._file_info = null;

        this._send_header( "ZSKIP" );

        return ret;
    }

    _send_ZRINIT() {
        this._send_header( "ZRINIT", ZRINIT_FLAGS );
    }

    _consume_ZFIN() {
        this._got_ZFIN = true;
        this._send_header( "ZFIN" );
    }

    _consume_ZEOF(header) {
        if (this._file_offset !== header.get_offset()) {
            throw( "ZEOF offset mismatch; unimplemented (local: " + this._file_offset + "; ZEOF: " + header.get_offset() + ")" );
        }

        this._on_file_end();

        //Preserve these two so that file_end callbacks
        //will have the right information.
        this._file_info = null;
        this._current_transfer = null;
    }

    _consume_ZDATA(header) {
        if ( this._file_offset === header.get_offset() ) {
            this._offset_ok = true;
        }
        else {
            throw "Error correction is unimplemented.";
        }
    }

    _send_ZRPOS() {
        this._send_header( "ZRPOS", this._file_offset );
    }

    //----------------------------------------------------------------------
    //events

    _on_file_end() {
        this._Happen("file_end");

        if (this._current_transfer) {
            this._current_transfer._Happen("complete");
            this._current_transfer = null;
        }
    }

    _on_data_in(subpacket) {
        this._Happen("data_in", subpacket);

        if (this._current_transfer) {
            this._current_transfer._Happen("input", subpacket.get_payload());
        }
    }
}

Object.assign(
    Zmodem.Session.Receive.prototype,
    {
        type: "receive",
    }
);

//----------------------------------------------------------------------

/**
 * @typedef {Object} FileDetails
 *
 * @property {string} name - The name of the file.
 *
 * @property {number} [size] - The file size, in bytes.
 *
 * @property {number} [mode] - The file mode (e.g., 0100644).
 *
 * @property {Date|number} [mtime] - The file’s modification time.
 *  When expressed as a number, the unit is epoch seconds.
 *
 * @property {number} [files_remaining] - Inclusive of the current file,
 *  so this value is never less than 1.
 *
 * @property {number} [bytes_remaining] - Inclusive of the current file.
 */

/**
 * Common methods for Transfer and Offer objects.
 *
 * @mixin
 */
var Transfer_Offer_Mixin = {
    /**
     * Returns the file details object.
     * @returns {FileDetails} `mtime` is a Date.
     */
    get_details: function get_details() {
        return Object.assign( {}, this._file_info );
    },

    /**
     * Returns a parse of the ZFILE header’s payload.
     *
     * @returns {Object} Members are:
     *
     * - `conversion` (string | undefined)
     * - `management` (string | undefined)
     * - `transfer` (string | undefined)
     * - `sparse` (boolean)
     */
    get_options: function get_options() {
        return Object.assign( {}, this._zfile_opts );
    },

    /**
     * Returns the offset based on the last transferred chunk.
     * @returns {number} The file offset (i.e., number of bytes after
     *  the start of the file).
     */
    get_offset: function get_offset() {
        return this._file_offset;
    },
};

/**
 * A class to represent a sender’s interaction with a single file
 * transfer within a batch. When a receiver accepts an offer, the
 * Session instantiates this class and passes the instance as the
 * promise resolution from send_offer().
 *
 * @mixes Transfer_Offer_Mixin
 */
class Transfer {

    /**
     * Not called directly.
     */
    constructor(file_info, offset, send_func, end_func) {
        this._file_info = file_info;
        this._file_offset = offset || 0;

        this._send = send_func;
        this._end = end_func;
    }

    /**
     * Send a (non-terminal) piece of the file.
     *
     * @param { number[] | Uint8Array } array_like - The bytes to send.
     */
    send(array_like) {
        this._send(array_like);
        this._file_offset += array_like.length;
    }

    /**
     * Complete the file transfer.
     *
     * @param { number[] | Uint8Array } [array_like] - The last bytes to send.
     *
     * @return { Promise } Resolves when the receiver has indicated
     *      acceptance of the end of the file transfer.
     */
    end(array_like) {
        var ret = this._end(array_like || []);
        if (array_like) this._file_offset += array_like.length;
        return ret;
    }
}
Object.assign( Transfer.prototype, Transfer_Offer_Mixin );

/**
 * A class to represent a receiver’s interaction with a single file
 * transfer offer within a batch. There is functionality here to
 * skip or accept offered files and either to spool the packet
 * payloads or to handle them yourself.
 *
 * @mixes Transfer_Offer_Mixin
 */
class Offer extends _Eventer {

    /**
     * Not called directly.
     */
    constructor(zfile_opts, file_info, accept_func, skip_func) {
        super();

        this._zfile_opts = zfile_opts;
        this._file_info = file_info;

        this._accept_func = accept_func;
        this._skip_func = skip_func;

        this._Add_event("input");
        this._Add_event("complete");

        //Register this first so that application handlers receive
        //the updated offset.
        this.on("input", this._input_handler);
    }

    _verify_not_skipped() {
        if (this._skipped) {
            throw new Zmodem.Error("Already skipped!");
        }
    }

    /**
     * Tell the sender that you don’t want the offered file.
     *
     * You can send this in lieu of `accept()` or after it, e.g.,
     * if you find that the transfer is taking too long. Note that,
     * if you `skip()` after you `accept()`, you’ll likely have to
     * wait for buffers to clear out.
     *
     */
    skip() {
        this._verify_not_skipped();
        this._skipped = true;

        return this._skip_func.apply(this, arguments);
    }

    /**
     * Tell the sender to send the offered file.
     *
     * @param {Object} [opts] - Can be:
     * @param {string} [opts.oninput=spool_uint8array] - Can be:
     *
     * - `spool_uint8array`: Stores the ZMODEM
     *     packet payloads as Uint8Array instances.
     *     This makes for an easy transition to a Blob,
     *     which JavaScript can use to save the file to disk.
     *
     * - `spool_array`: Stores the ZMODEM packet payloads
     *     as Array instances. Each value is an octet value.
     *
     * - (function): A handler that receives each payload
     *     as it arrives. The Offer object does not store
     *     the payloads internally when thus configured.
     *
     * @return { Promise } Resolves when the file is fully received.
     *      If the Offer has been spooling
     *      the packet payloads, the promise resolves with an Array
     *      that contains those payloads.
     */
    accept(opts) {
        this._verify_not_skipped();

        if (this._accepted) {
            throw new Zmodem.Error("Already accepted!");
        }
        this._accepted = true;

        if (!opts) opts = {};

        this._file_offset = opts.offset || 0;

        switch (opts.on_input) {
            case null:
            case undefined:
            case "spool_array":
            case DEFAULT_RECEIVE_INPUT_MODE:    //default
                this._spool = [];
                break;
            default:
                if (typeof opts.on_input !== "function") {
                    throw "Invalid “on_input”: " + opts.on_input;
                }
        }

        this._input_handler_mode = opts.on_input || DEFAULT_RECEIVE_INPUT_MODE;

        return this._accept_func(this._file_offset).then( this._get_spool.bind(this) );
    }

    _input_handler(payload) {
        this._file_offset += payload.length;

        if (typeof this._input_handler_mode === "function") {
            this._input_handler_mode(payload);
        }
        else {
            if (this._input_handler_mode === DEFAULT_RECEIVE_INPUT_MODE) {
                payload = new Uint8Array(payload);
            }

            //sanity
            else if (this._input_handler_mode !== "spool_array") {
                throw new Zmodem.Error("WTF?? _input_handler_mode = " + this._input_handler_mode);
            }

            this._spool.push(payload);
        }
    }

    _get_spool() {
        return this._spool;
    }
}
Object.assign( Offer.prototype, Transfer_Offer_Mixin );

//Curious that ZSINIT isn’t here … but, lsz sends it as hex.
const SENDER_BINARY_HEADER = {
    ZFILE: true,
    ZDATA: true,
};

/**
 * A class that encapsulates behavior for a ZMODEM sender.
 *
 * @extends Session
 */
Zmodem.Session.Send = class ZmodemSendSession extends Zmodem.Session {

    /**
     * Not called directly.
     */
    constructor(zrinit_hdr) {
        super();

        if (!zrinit_hdr) {
            throw "Need first header!";
        }
        else if (zrinit_hdr.NAME !== "ZRINIT") {
            throw("First header should be ZRINIT, not " + zrinit_hdr.NAME);
        }

        this._last_header_name = 'ZRINIT';

        //We don’t need to send crc32. Even if the other side can grok it,
        //there’s no point to sending it since, for now, we assume we’re
        //on a reliable connection, e.g., TCP. Ideally we’d just forgo
        //CRC checks completely, but ZMODEM doesn’t allow that.
        //
        //If we *were* to start using crc32, we’d update this every time
        //we send a header.
        this._subpacket_encode_func = 'encode16';

        this._zencoder = new Zmodem.ZDLE();

        this._consume_ZRINIT(zrinit_hdr);

        this._file_offset = 0;

        var zrqinit_count = 0;

        this._start_keepalive_on_set_sender = true;

        //lrzsz will send ZRINIT until it gets an offer. (keep-alive?)
        //It sends 4 additional ones after the initial ZRINIT and, if
        //no response is received, starts sending “C” (0x43, 67) as if to
        //try to downgrade to XMODEM or YMODEM.
        //var sess = this;
        //this._prepare_to_receive_ZRINIT( function keep_alive() {
        //    sess._prepare_to_receive_ZRINIT(keep_alive);
        //} );

        //queue up the ZSINIT flag to send -- but seems useless??

        /*
        Object.assign(
            this._on_evt,
            {
                file_received: [],
            },
        };
        */
    }

    /**
     * Sets the sender function. The first time this is called,
     * it will also initiate a keepalive using ZSINIT until the
     * first file is sent.
     *
     * @param {Function} func - The function to call.
     *  It will receive an Array with the relevant octets.
     *
     * @return {Session} The session object (for chaining).
     */
    set_sender(func) {
        super.set_sender(func);

        if (this._start_keepalive_on_set_sender) {
            this._start_keepalive_on_set_sender = false;
            this._start_keepalive();
        }

        return this;
    }

    //7.3.3 .. The sender also uses hex headers when they are
    //not followed by binary data subpackets.
    //
    //FG: … or when the header is ZSINIT? That’s what lrzsz does, anyway.
    //Then it sends a single NUL byte as the payload to an end_ack subpacket.
    _get_header_formatter(name) {
        return SENDER_BINARY_HEADER[name] ? "to_binary16" : "to_hex";
    }

    //In order to keep lrzsz from timing out, we send ZSINIT every 5 seconds.
    //Maybe make this configurable?
    _start_keepalive() {
        //if (this._keepalive_promise) throw "Keep-alive already started!";
        if (!this._keepalive_promise) {
            var sess = this;

            this._keepalive_promise = new Promise(function(resolve) {
                //console.log("SETTING KEEPALIVE TIMEOUT");
                sess._keepalive_timeout = setTimeout(resolve, KEEPALIVE_INTERVAL);
            }).then( function() {
                sess._next_header_handler = {
                    ZACK: function() {

                        //We’re going to need to ensure that the
                        //receiver is ready for all control characters
                        //to be escaped. If we’ve already sent a ZSINIT
                        //and gotten a response, then we know that that
                        //work is already done later on when we actually
                        //send an offer.
                        sess._got_ZSINIT_ZACK = true;
                    },
                };
                sess._send_ZSINIT();

                sess._keepalive_promise = null;
                sess._start_keepalive();
            });
        }
    }

    _stop_keepalive() {
        if (this._keepalive_promise) {
            //console.log("STOPPING KEEPALIVE");
            clearTimeout(this._keepalive_timeout);
            this._keep_alive_promise = null;
        }
    }

    _send_ZSINIT() {
        //See note at _ensure_receiver_escapes_ctrl_chars()
        //for why we have to pass ESCCTL.

        var zsinit_flags = [];
        if (this._zencoder.escapes_ctrl_chars()) {
            zsinit_flags.push("ESCCTL");
        }

        this._send_header_and_data(
            ["ZSINIT", zsinit_flags],
            [0],
            "end_ack"
        );
    }

    _consume_ZRINIT(hdr) {
        this._last_ZRINIT = hdr;

        if (hdr.get_buffer_size()) {
            throw( "Buffer size (" + hdr.get_buffer_size() + ") is unsupported!" );
        }

        if (!hdr.can_full_duplex()) {
            throw( "Half-duplex I/O is unsupported!" );
        }

        if (!hdr.can_overlap_io()) {
            throw( "Non-overlap I/O is unsupported!" );
        }

        if (hdr.escape_8th_bit()) {
            throw( "8-bit escaping is unsupported!" );
        }

        if (FORCE_ESCAPE_CTRL_CHARS) {
            this._zencoder.set_escape_ctrl_chars(true);
            if (!hdr.escape_ctrl_chars()) {
                console.debug("Peer didn’t request escape of all control characters. Will send ZSINIT to force recognition of escaped control characters.");
            }
        }
        else {
            this._zencoder.set_escape_ctrl_chars(hdr.escape_ctrl_chars());
        }
    }

    //https://stackoverflow.com/questions/23155939/missing-0xf-and-0x16-when-binary-data-through-virtual-serial-port-pair-created-b
    //^^ Because of that, we always escape control characters.
    //The alternative would be that lrz would never receive those
    //two bytes from zmodem.js.
    _ensure_receiver_escapes_ctrl_chars() {
        var promise;

        var needs_ZSINIT = !this._last_ZRINIT.escape_ctrl_chars() && !this._got_ZSINIT_ZACK;

        if (needs_ZSINIT) {
            var sess = this;
            promise = new Promise( function(res) {
                sess._next_header_handler = {
                    ZACK: (hdr) => {
                        res();
                    },
                };
                sess._send_ZSINIT();
            } );
        }
        else {
            promise = Promise.resolve();
        }

        return promise;
    }

    _convert_params_to_offer_payload_array(params) {
        params = Zmodem.Validation.offer_parameters(params);

        var subpacket_payload = params.name + "\x00";

        var subpacket_space_pieces = [
            (params.size || 0).toString(10),
            params.mtime ? params.mtime.toString(8) : "0",
            params.mode ? (0x8000 | params.mode).toString(8) : "0",
            "0",    //serial
        ];

        if (params.files_remaining) {
            subpacket_space_pieces.push( params.files_remaining );

            if (params.bytes_remaining) {
                subpacket_space_pieces.push( params.bytes_remaining );
            }
        }

        subpacket_payload += subpacket_space_pieces.join(" ");
        return this._string_to_octets(subpacket_payload);
    }

    /**
     * Send an offer to the receiver.
     *
     * @param {FileDetails} params - All about the file you want to transfer.
     *
     * @returns {Promise} If the receiver accepts the offer, then the
     * resolution is a Transfer object; otherwise the resolution is
     * undefined.
     */
    send_offer(params) {
        if (Zmodem.DEBUG) {
            console.debug("SENDING OFFER", params);
        }

        if (!params) throw "need file params!";

        if (this._sending_file) throw "Already sending file!";

        var payload_array = this._convert_params_to_offer_payload_array(params);

        this._stop_keepalive();

        var sess = this;

        function zrpos_handler_setter_func() {
            sess._next_header_handler = {

                // The receiver may send ZRPOS in at least two cases:
                //
                // 1) A malformed subpacket arrived, so we need to
                // “rewind” a bit and continue from the receiver’s
                // last-successful location in the file.
                //
                // 2) The receiver hasn’t gotten any data for a bit,
                // so it sends ZRPOS as a “ping”.
                //
                // Case #1 shouldn’t happen since zmodem.js requires a
                // reliable transport. Case #2, though, can happen due
                // to either normal network congestion or errors in
                // implementation. In either case, there’s nothing for
                // us to do but to ignore the ZRPOS, with an optional
                // warning.
                //
                ZRPOS: function(hdr) {
                    if (Zmodem.DEBUG) {
                        console.warn("Mid-transfer ZRPOS … implementation error?");
                    }

                    zrpos_handler_setter_func();
                },
            };
        };

        var doer_func = function() {

            //return Promise object that is fulfilled when the ZRPOS or ZSKIP arrives.
            //The promise value is the byte offset, or undefined for ZSKIP.
            //If ZRPOS arrives, then send ZDATA(0) and set this._sending_file.
            var handler_setter_promise = new Promise( function(res) {
                sess._next_header_handler = {
                    ZSKIP: function() {
                        sess._start_keepalive();
                        res();
                    },
                    ZRPOS: function(hdr) {
                        sess._sending_file = true;

                        zrpos_handler_setter_func();

                        res(
                            new Transfer(
                                params,
                                hdr.get_offset(),
                                sess._send_interim_file_piece.bind(sess),
                                sess._end_file.bind(sess)
                            )
                        );
                    },
                };
            } );

            sess._send_header_and_data( ["ZFILE"], payload_array, "end_ack" );

            delete sess._sent_ZDATA;

            return handler_setter_promise;
        };

        if (FORCE_ESCAPE_CTRL_CHARS) {
            return this._ensure_receiver_escapes_ctrl_chars().then(doer_func);
        }

        return doer_func();
    }

    _send_header_and_data( hdr_name_and_args, data_arr, frameend ) {
        var bytes_hdr = this._create_header_bytes(hdr_name_and_args);

        var data_bytes = this._build_subpacket_bytes(data_arr, frameend);

        bytes_hdr[0].push.apply( bytes_hdr[0], data_bytes );

        if (Zmodem.DEBUG) {
            this._log_header( "SENDING HEADER", bytes_hdr[1] );
            console.debug( this.type, "-- HEADER PAYLOAD:", frameend, data_bytes.length );
        }

        this._sender( bytes_hdr[0] );

        this._last_sent_header = bytes_hdr[1];
    }

    _build_subpacket_bytes( bytes_arr, frameend ) {
        var subpacket = Zmodem.Subpacket.build(bytes_arr, frameend);

        return subpacket[this._subpacket_encode_func]( this._zencoder );
    }

    _build_and_send_subpacket( bytes_arr, frameend ) {
        this._sender( this._build_subpacket_bytes(bytes_arr, frameend) );
    }

    _string_to_octets(string) {
        if (!this._textencoder) {
            this._textencoder = new Zmodem.Text.Encoder();
        }

        var uint8arr = this._textencoder.encode(string);
        return Array.prototype.slice.call(uint8arr);
    }

    /*
    Potential future support for responding to ZRPOS:
    send_file_offset(offset) {
    }
    */

    /*
        Sending logic works thus:
            - ASSUME the receiver can overlap I/O (CANOVIO)
                (so fail if !CANFDX || !CANOVIO)
            - Sender opens the firehose … all ZCRCG (!end/!ack)
                until the end, when we send a ZCRCE (end/!ack)
                NB: try 8k/32k/64k chunk sizes? Looks like there’s
                no need to change the packet otherwise.
    */
    //TODO: Put this on a Transfer object similar to what Receive uses?
    _send_interim_file_piece(bytes_obj) {

        //We don’t ask the receiver to confirm because there’s no need.
        this._send_file_part(bytes_obj, "no_end_no_ack");

        //This pattern will allow
        //error-correction without buffering the entire stream in JS.
        //For now the promise is always resolved, but in the future we
        //can make it only resolve once we’ve gotten acknowledgement.
        return Promise.resolve();
    }

    _ensure_we_are_sending() {
        if (!this._sending_file) throw "Not sending a file currently!";
    }

    //This resolves once we receive ZEOF.
    _end_file(bytes_obj) {
        this._ensure_we_are_sending();

        //Is the frame-end-ness of this last packet redundant
        //with the ZEOF packet?? - No. It signals the receiver that
        //the next thing to expect is a header, not a packet.

        //no-ack, following lrzsz’s example
        this._send_file_part(bytes_obj, "end_no_ack");

        var sess = this;

        //Register this before we send ZEOF in case of local round-trip.
        //(Basically just for synchronous testing, but.)
        var ret = new Promise( function(res) {
            //console.log("UNSETTING SENDING FLAG");
            sess._sending_file = false;
            sess._prepare_to_receive_ZRINIT(res);
        } );

        this._send_header( "ZEOF", this._file_offset );

        this._file_offset = 0;

        return ret;
    }

    //Called at the beginning of our session
    //and also when we’re done sending a file.
    _prepare_to_receive_ZRINIT(after_consume) {
        this._next_header_handler = {
            ZRINIT: function(hdr) {
                this._consume_ZRINIT(hdr);
                if (after_consume) after_consume();
            },
        };
    }

    /**
     * Signal to the receiver that the ZMODEM session is wrapping up.
     *
     * @returns {Promise} Resolves when the receiver has responded to
     * our signal that the session is over.
     */
    close() {
        var ok_to_close = (this._last_header_name === "ZRINIT")
        if (!ok_to_close) {
            ok_to_close = (this._last_header_name === "ZSKIP");
        }
        if (!ok_to_close) {
            ok_to_close = (this._last_sent_header.name === "ZSINIT") &&  (this._last_header_name === "ZACK");
        }

        if (!ok_to_close) {
            throw( "Can’t close; last received header was “" + this._last_header_name + "”" );
        }

        var sess = this;

        var ret = new Promise( function(res, rej) {
            sess._next_header_handler = {
                ZFIN: function() {
                    sess._sender( OVER_AND_OUT );
                    sess._sent_OO = true;
                    sess._on_session_end();
                    res();
                },
            };
        } );

        this._send_header("ZFIN");

        return ret;
    }

    _has_ended() {
        return this.aborted() || !!this._sent_OO;
    }

    _send_file_part(bytes_obj, final_packetend) {
        if (!this._sent_ZDATA) {
            this._send_header( "ZDATA", this._file_offset );
            this._sent_ZDATA = true;
        }

        var obj_offset = 0;

        var bytes_count = bytes_obj.length;

        //We have to go through at least once in event of an
        //empty buffer, e.g., an empty end_file.
        while (true) {
            var chunk_size = Math.min(obj_offset + MAX_CHUNK_LENGTH, bytes_count) - obj_offset;

            var at_end = (chunk_size + obj_offset) >= bytes_count;

            var chunk = bytes_obj.slice( obj_offset, obj_offset + chunk_size );
            if (!(chunk instanceof Array)) {
                chunk = Array.prototype.slice.call(chunk);
            }

            this._build_and_send_subpacket(
                chunk,
                at_end ? final_packetend : "no_end_no_ack"
            );

            this._file_offset += chunk_size;
            obj_offset += chunk_size;

            if (obj_offset >= bytes_count) break;
        }
    }

    _consume_first() {
        if (!this._parse_and_consume_header()) {

            //When the ZMODEM receive program starts, it immediately sends
            //a ZRINIT header to initiate ZMODEM file transfers, or a
            //ZCHALLENGE header to verify the sending program. The receive
            //program resends its header at response time (default 10 second)
            //intervals for a suitable period of time (40 seconds total)
            //before falling back to YMODEM protocol.
            if (this._input_buffer.join() === "67") {
                throw "Receiver has fallen back to YMODEM.";
            }
        }
    }

    _on_session_end() {
        this._stop_keepalive();
        super._on_session_end();
    }
}

Object.assign(
    Zmodem.Session.Send.prototype,
    {
        type: "send",
    }
);
