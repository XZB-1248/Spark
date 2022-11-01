"use strict";

var Zmodem = module.exports;

//TODO: Make this usable without require.js or what not.
window.Zmodem = Zmodem;

Object.assign(
    Zmodem,
    require("./zmodem")
);

function _check_aborted(session) {
    if (session.aborted()) {
        throw new Zmodem.Error("aborted");
    }
}

/** Browser-specific tools
 *
 * @exports Browser
 */
Zmodem.Browser = {

    /**
     * Send a batch of files in sequence. The session is left open
     * afterward, which allows for more files to be sent if desired.
     *
     * @param {Zmodem.Session} session - The send session
     *
     * @param {FileList|Array} files - A list of File objects
     *
     * @param {Object} [options]
     * @param {Function} [options.on_offer_response] - Called when an
     * offer response arrives. Arguments are:
     *
     * - (File) - The File object that corresponds to the offer.
     * - (Transfer|undefined) - If the receiver accepts the offer, then
     * this is a Transfer object; otherwise it’s undefined.
     *
     * @param {Function} [options.on_progress] - Called immediately
     * after a chunk of a file is sent. Arguments are:
     *
     * - (File) - The File object that corresponds to the file.
     * - (Transfer) - The Transfer object for the current transfer.
     * - (Uint8Array) - The chunk of data that was just loaded from disk
     * and sent to the receiver.
     *
     * @param {Function} [options.on_file_complete] - Called immediately
     * after the last file packet is sent. Arguments are:
     *
     * - (File) - The File object that corresponds to the file.
     * - (Transfer) - The Transfer object for the now-completed transfer.
     *
     * @return {Promise} A Promise that fulfills when the batch is done.
     *      Note that skipped files are not considered an error condition.
     */
    send_files: function send_files(session, files, options) {
        if (!options) options = {};

        //Populate the batch in reverse order to simplify sending
        //the remaining files/bytes components.
        var batch = [];
        var total_size = 0;
        for (var f=files.length - 1; f>=0; f--) {
            var fobj = files[f];
            total_size += fobj.size;
            batch[f] = {
                obj: fobj,
                name: fobj.name,
                size: fobj.size,
                mtime: new Date(fobj.lastModified),
                files_remaining: files.length - f,
                bytes_remaining: total_size,
            };
        }

        var file_idx = 0;
        function promise_callback() {
            var cur_b = batch[file_idx];

            if (!cur_b) {
                return Promise.resolve(); //batch done!
            }

            file_idx++;

            return session.send_offer(cur_b).then( function after_send_offer(xfer) {
                if (options.on_offer_response) {
                    options.on_offer_response(cur_b.obj, xfer);
                }

                if (xfer === undefined) {
                    return promise_callback();   //skipped
                }

                return new Promise( function(res) {
                    var reader = new FileReader();

                    //This really shouldn’t happen … so let’s
                    //blow up if it does.
                    reader.onerror = function reader_onerror(e) {
                        console.error("file read error", e);
                        throw("File read error: " + e);
                    };

                    var piece;
                    reader.onprogress = function reader_onprogress(e) {

                        //Some browsers (e.g., Chrome) give partial returns,
                        //while others (e.g., Firefox) don’t.
                        if (e.target.result) {
                            piece = new Uint8Array(e.target.result, xfer.get_offset())

                            _check_aborted(session);

                            xfer.send(piece);

                            if (options.on_progress) {
                                options.on_progress(cur_b.obj, xfer, piece);
                            }
                        }
                    };

                    reader.onload = function reader_onload(e) {
                        piece = new Uint8Array(e.target.result, xfer, piece)

                        _check_aborted(session);

                        xfer.end(piece).then( function() {
                            if (options.on_progress && piece.length) {
                                options.on_progress(cur_b.obj, xfer, piece);
                            }

                            if (options.on_file_complete) {
                                options.on_file_complete(cur_b.obj, xfer);
                            }

                            //Resolve the current file-send promise with
                            //another promise. That promise resolves immediately
                            //if we’re done, or with another file-send promise
                            //if there’s more to send.
                            res( promise_callback() );
                        } );
                    };

                    reader.readAsArrayBuffer(cur_b.obj);
                } );
            } );
        }

        return promise_callback();
    },

    /**
     * Prompt a user to save the given packets as a file by injecting an
     * `<a>` element (with `display: none` styling) into the page and
     * calling the element’s `click()`
     * method. The element is removed immediately after.
     *
     * @param {Array} packets - Same as the first argument to [Blob’s constructor](https://developer.mozilla.org/en-US/docs/Web/API/Blob).
     * @param {string} name - The name to give the file.
     */
    save_to_disk: function save_to_disk(packets, name) {
        var blob = new Blob(packets);
        var url = URL.createObjectURL(blob);

        var el = document.createElement("a");
        el.style.display = "none";
        el.href = url;
        el.download = name;

        //It seems like a security problem that this actually works;
        //I’d think there would need to be some confirmation before
        //a browser could save arbitrarily many bytes onto the disk.
        //But, hey.
        el.click();
        setTimeout(() => URL.revokeObjectURL(url), 10000);
    },
};
