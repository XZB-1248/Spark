"use strict";

var Zmodem = module.exports;

Object.assign(
    Zmodem,
    require("./zmlib"),
    require("./zsession")
);

const
    MIN_ZM_HEX_START_LENGTH = 20,
    MAX_ZM_HEX_START_LENGTH = 21,

    // **, ZDLE, 'B0'
    //ZRQINIT’s next byte will be '0'; ZRINIT’s will be '1'.
    COMMON_ZM_HEX_START = [ 42, 42, 24, 66, 48 ],

    SENTRY_CONSTRUCTOR_REQUIRED_ARGS = [
        "to_terminal",
        "on_detect",
        "on_retract",
        "sender",
    ],

    ASTERISK = 42
;

/**
 * An instance of this object is passed to the Sentry’s on_detect
 * callback each time the Sentry object sees what looks like the
 * start of a ZMODEM session.
 *
 * Note that it is possible for a detection to be “retracted”
 * if the Sentry consumes bytes afterward that are not ZMODEM.
 * When this happens, the Sentry’s `retract` event will fire,
 * after which the Detection object is no longer usable.
 */
class Detection {

    /**
     * Not called directly.
     */
    constructor(session_type, accepter, denier, checker) {

        //confirm() - user confirms that ZMODEM is desired
        this._confirmer = accepter;

        //deny() - user declines ZMODEM; send abort sequence
        //
        //TODO: It might be ideal to forgo the session “peaceably”,
        //i.e., such that the peer doesn’t end in error. That’s
        //possible if we’re the sender, we accept the session,
        //then we just send a close(), but it doesn’t seem to be
        //possible for a receiver. Thus, let’s just leave it so
        //it’s at least consistent (and simpler, too).
        this._denier = denier;

        this._is_valid = checker;

        this._session_type = session_type;
    }

    /**
     * Confirm that the detected ZMODEM sequence indicates the
     * start of a ZMODEM session.
     *
     * @return {Session} The ZMODEM Session object (i.e., either a
     *  Send or Receive instance).
     */
    confirm() {
        return this._confirmer.apply(this, arguments);
    }

    /**
     * Tell the Sentry that the detected bytes sequence is
     * **NOT** intended to be the start of a ZMODEM session.
     */
    deny() {
        return this._denier.apply(this, arguments);
    }

    /**
     * Tells whether the Detection is still valid; i.e., whether
     * the Sentry has `consume()`d bytes that invalidate the
     * Detection.
     *
     * @returns {boolean} Whether the Detection is valid.
     */
    is_valid() {
        return this._is_valid.apply(this, arguments);
    }

    /**
     * Gives the session’s role.
     *
     * @returns {string} One of:
     * - `receive`
     * - `send`
     */
    get_session_role() { return this._session_type }
}

/**
 * Class that parses an input stream for the beginning of a
 * ZMODEM session. We look for the tell-tale signs
 * of a ZMODEM transfer and allow the client to determine whether
 * it’s really ZMODEM or not.
 *
 * This is the “mother” class for zmodem.js;
 * all other class instances are created, directly or indirectly,
 * by an instance of this class.
 *
 * This logic is not unlikely to need tweaking, and it can never
 * be fully bulletproof; if it could be bulletproof it would be
 * simpler since there wouldn’t need to be the .confirm()/.deny()
 * step.
 *
 * One thing you could do to make things a bit simpler *is* just
 * to make that assumption for your users--i.e., to .confirm()
 * Detection objects automatically. That’ll be one less step
 * for the user, but an unaccustomed user might find that a bit
 * confusing. It’s also then possible to have a “false positive”:
 * a text stream that contains a ZMODEM initialization string but
 * isn’t, in fact, meant to start a ZMODEM session.
 *
 * Workflow:
 *  - parse all input with .consume(). As long as nothing looks
 *      like ZMODEM, all the traffic will go to to_terminal().
 *
 *  - when a “tell-tale” sequence of bytes arrives, we create a
 *      Detection object and pass it to the “on_detect” handler.
 *
 *  - Either .confirm() or .deny() with the Detection object.
 *      This is the user’s chance to say, “yeah, I know those
 *      bytes look like ZMODEM, but they’re not. So back off!”
 *
 *      If you .confirm(), the Session object is returned, and
 *      further input that goes to the Sentry’s .consume() will
 *      go to the (now-active) Session object.
 *
 *  - Sometimes additional traffic arrives that makes it apparent
 *      that no ZMODEM session is intended to start; in this case,
 *      the Sentry marks the Detection as “stale” and calls the
 *      `on_retract` handler. Any attempt from here to .confirm()
 *      on the Detection object will prompt an exception.
 *
 *      (This “retraction” behavior will only happen prior to
 *      .confirm() or .deny() being called on the Detection object.
 *      Beyond that point, either the Session has to deal with the
 *      “garbage”, or it’s back to the terminal anyway.
 *
 *  - Once the Session object is done, the Sentry will again send
 *      all traffic to to_terminal().
 */
Zmodem.Sentry = class ZmodemSentry {

    /**
     * Invoked directly. Creates a new Sentry that inspects all
     * traffic before it goes to the terminal.
     *
     * @param {Object} options - The Sentry parameters
     *
     * @param {Function} options.to_terminal - Handler that sends
     * traffic to the terminal object. Receives an iterable object
     * (e.g., an Array) that contains octet numbers.
     *
     * @param {Function} options.on_detect - Handler for new
     * detection events. Receives a new Detection object.
     *
     * @param {Function} options.on_retract - Handler for retraction
     * events. Receives no input.
     *
     * @param {Function} options.sender - Handler that sends traffic to
     * the peer. If, for example, your application uses WebSocket to talk
     * to the peer, use this to send data to the WebSocket instance.
     */
    constructor(options) {
        if (!options) throw "Need options!";

        var sentry = this;
        SENTRY_CONSTRUCTOR_REQUIRED_ARGS.forEach( function(arg) {
            if (!options[arg]) {
                throw "Need “" + arg + "”!";
            }
            sentry["_" + arg] = options[arg];
        } );

        this._cache = [];
    }

    _after_session_end() {
        this._zsession = null;
    }

    /**
     * “Consumes” a piece of input:
     *
     *  - If there is no active or pending ZMODEM session, the text is
     *      all output. (This is regardless of whether we’ve got a new
     *      Detection.)
     *
     *  - If there is no active ZMODEM session and the input **ends** with
     *      a ZRINIT or ZRQINIT, then a new Detection object is created,
     *      and it is passed to the “on_detect” function.
     *      If there was another pending Detection object, it is retracted.
     *
     *  - If there is no active ZMODEM session and the input does NOT end
     *      with a ZRINIT or ZRQINIT, then any pending Detection object is
     *      retracted.
     *
     *  - If there is an active ZMODEM session, the input is passed to it.
     *      Any non-ZMODEM data (i.e., “garbage”) parsed from the input
     *      is sent to output.
     *      If the ZMODEM session ends, any post-ZMODEM part of the input
     *      is sent to output.
     *
     *  @param {number[] | ArrayBuffer} input - Octets to parse as input.
     */
    consume(input) {
        if (!(input instanceof Array)) {
            input = Array.prototype.slice.call( new Uint8Array(input) );
        }

        if (this._zsession) {
            var session_before_consume = this._zsession;

            session_before_consume.consume(input);

            if (session_before_consume.has_ended()) {
                if (session_before_consume.type === "receive") {
                    input = session_before_consume.get_trailing_bytes();
                }
                else {
                    input = [];
                }
            }
            else return;
        }

        var new_session = this._parse(input);
        var to_terminal = input;

        if (new_session) {
            let replacement_detect = !!this._parsed_session;

            if (replacement_detect) {
                //no terminal output if the new session is of the
                //same type as the old
                if (this._parsed_session.type === new_session.type) {
                    to_terminal = [];
                }

                this._on_retract();
            }

            this._parsed_session = new_session;

            var sentry = this;

            function checker() {
                return sentry._parsed_session === new_session;
            }

            //This runs with the Sentry object as the context.
            function accepter() {
                if (!this.is_valid()) {
                    throw "Stale ZMODEM session!";
                }

                new_session.on("garbage", sentry._to_terminal);

                new_session.on(
                    "session_end",
                    sentry._after_session_end.bind(sentry)
                );

                new_session.set_sender(sentry._sender);

                delete sentry._parsed_session;

                return sentry._zsession = new_session;
            };

            function denier() {
                if (!this.is_valid()) return;
            };

            this._on_detect( new Detection(
                new_session.type,
                accepter,
                this._send_abort.bind(this),
                checker
            ) );
        }
        else {
            /*
            if (this._parsed_session) {
                this._session_stale_because = 'Non-ZMODEM output received after ZMODEM initialization.';
            }
            */

            var expired_session = this._parsed_session;

            this._parsed_session = null;

            if (expired_session) {

                //If we got a single “C” after parsing a session,
                //that means our peer is trying to downgrade to YMODEM.
                //That won’t work, so we just send the ABORT_SEQUENCE
                //right away.
                if (to_terminal.length === 1 && to_terminal[0] === 67) {
                    this._send_abort();
                }

                this._on_retract();
            }
        }

        this._to_terminal(to_terminal);
    }

    /**
     * @return {Session|null} The sentry’s current Session object, or
     *      null if there is none.
     */
    get_confirmed_session() {
        return this._zsession || null;
    }

    _send_abort() {
        this._sender( Zmodem.ZMLIB.ABORT_SEQUENCE );
    }

    /**
     * Parse an input stream and decide how much of it goes to the
     * terminal or to a new Session object.
     *
     * This will accommodate input strings that are fragmented
     * across calls to this function; e.g., if you send the first
     * two bytes at the end of one parse() call then send the rest
     * at the beginning of the next, parse() will recognize it as
     * the beginning of a ZMODEM session.
     *
     * In order to keep from blocking any actual useful data to the
     * terminal in real-time, this will send on the initial
     * ZRINIT/ZRQINIT bytes to the terminal. They’re meant to go to the
     * terminal anyway, so that should be fine.
     *
     * @private
     *
     * @param {Array|Uint8Array} array_like - The input bytes.
     *      Each member should be a number between 0 and 255 (inclusive).
     *
     * @return {Array} A two-member list:
     *      0) the bytes that should be printed on the terminal
     *      1) the created Session object (if any)
     */
    _parse(array_like) {
        var cache = this._cache;

        cache.push.apply( cache, array_like );

        while (true) {
            let common_hex_at = Zmodem.ZMLIB.find_subarray( cache, COMMON_ZM_HEX_START );
            if (-1 === common_hex_at) break;

            let before_common_hex = cache.splice(0, common_hex_at);
            let zsession;
            try {
                zsession = Zmodem.Session.parse(cache);
            } catch(err) {     //ignore errors
                //console.log(err);
            }

            if (!zsession) break;

            //Don’t need to parse the trailing XON.
            if ((cache.length === 1) && (cache[0] === Zmodem.ZMLIB.XON)) {
                cache.shift();
            }

            //If there are still bytes in the cache,
            //then we don’t have a ZMODEM session. This logic depends
            //on the sender only sending one initial header.
            return cache.length ? null : zsession;
        }

        cache.splice( MAX_ZM_HEX_START_LENGTH );

        return null;
    }
}
