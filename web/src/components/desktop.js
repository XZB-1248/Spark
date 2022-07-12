import React, {useCallback, useEffect, useState} from 'react';
import {ab2str, formatSize, genRandHex, getBaseURL, translate, ws2ua} from "../utils/utils";
import i18n from "../locale/locale";
import DraggableModal from "./modal";
import CryptoJS from "crypto-js";
import {Button, message} from "antd";
import {FullscreenOutlined, ReloadOutlined} from "@ant-design/icons";
import "./desktop.css";

let ws = null;
let ctx = null;
let conn = false;
let canvas = null;
let secret = null;
let ticker = 0;
let frames = 0;
let bytes = 0;
let ticks = 0;
let title = i18n.t('desktop');
function ScreenModal(props) {
    const [bandwidth, setBandwidth] = useState(0);
    const [fps, setFps] = useState(0);
    const canvasRef = useCallback((e) => {
        if (e && props.visible && !conn) {
            canvas = e;
            initCanvas(canvas);
            construct(canvas);
        }
    }, [props]);
    useEffect(() => {
        if (props.visible) {
            secret = CryptoJS.enc.Hex.parse(genRandHex(32));
        } else {
            if (ws && conn) {
                clearInterval(ticker);
                ws.close();
                conn = false;
            }
        }
    }, [props.visible, props.device]);

    function initCanvas() {
        if (!canvas) return;
        ctx = canvas.getContext('2d');
    }
    function construct() {
        if (ctx !== null) {
            if (ws !== null && conn) {
                ws.close();
            }
            ws = new WebSocket(getBaseURL(true, `api/device/desktop?device=${props.device.id}&secret=${secret}`));
            ws.binaryType = 'arraybuffer';
            ws.onopen = () => {
                conn = true;
            }
            ws.onmessage = (e) => {
                parseBlocks(e.data, canvas, ctx);
            };
            ws.onclose = () => {
                if (conn) {
                    conn = false;
                    message.warn(i18n.t('disconnected'));
                }
            };
            ws.onerror = (e) => {
                console.error(e);
                if (conn) {
                    conn = false;
                    message.warn(i18n.t('disconnected'));
                } else {
                    message.warn(i18n.t('connectFailed'));
                }
            };
            clearInterval(ticker);
            ticker = setInterval(() => {
                setBandwidth(bytes);
                bytes = 0;
                setFps(frames);
                frames = 0;
                ticks++;
                if (ticks > 10 && conn) {
                    ticks = 0;
                    ws.send(encrypt({
                        act: 'ping'
                    }, secret));
                }
            }, 1000);
        }
    }
    function fullScreen() {
        try {
            canvas.requestFullscreen();
        } catch {}
        try {
            canvas.webkitRequestFullscreen();
        } catch {}
        try {
            canvas.mozRequestFullScreen();
        } catch {}
        try {
            canvas.msRequestFullscreen();
        } catch {}
    }
    function refresh() {
        if (canvas && props.visible) {
            if (!conn) {
                canvas.width = 1920;
                canvas.height = 1080;
                initCanvas(canvas);
                construct(canvas);
            } else {
                ws.send(encrypt({
                    act: 'getDesktop'
                }, secret));
            }
        }
    }

    function parseBlocks(ab, canvas, canvasCtx) {
        ab = ab.slice(5);
        let dv = new DataView(ab);
        let op = dv.getUint8(0);
        if (op === 3) {
            handleJSON(ab.slice(1));
            return;
        }
        if (op === 2) {
            canvas.width = dv.getUint16(1, false);
            canvas.height = dv.getUint16(3, false);
            return;
        }
        bytes += ab.byteLength;
        if (op === 0) { // 0 means this is first part of a frame.
            frames++;
        }
        let offset = 1;
        while (offset < ab.byteLength) {
            // let it = dv.getUint16(offset + 0, false); // image type
            // let bw = dv.getUint16(offset + 8, false); // image block width
            // let bh = dv.getUint16(offset + 10, false); // image block height
            let len = dv.getUint16(offset + 2, false); // image block length
            updateImage(ab.slice(offset, offset + len + 12), canvasCtx);
            offset += len + 12;
        }
    }
    function updateImage(ab, canvasCtx) {
        let dv = new DataView(ab);
        // let bl = dv.getUint16(2, false); // block length without header.
        let it = dv.getUint16(0, false); // image type: 0: raw, 1: jpg.
        let dx = dv.getUint16(4, false);
        let dy = dv.getUint16(6, false);
        let bw = dv.getUint16(8, false);
        let bh = dv.getUint16(10, false);
        ab = ab.slice(12);
        if (it === 0) {
            canvasCtx.putImageData(new ImageData(new Uint8ClampedArray(ab), bw, bh), dx, dy);
        } else {
            createImageBitmap(new Blob([ab]), 0, 0, bw, bh)
                .then((ib) => {
                    canvasCtx.drawImage(ib, dx, dy);
                });
        }
    }
    function handleJSON(ab) {
        let data = decrypt(ab, secret);
        try {
            data = JSON.parse(data);
        } catch (_) {}
        if (data?.act === 'warn') {
            message.warn(data.msg ? translate(data.msg) : i18n.t('unknownError'));
            return;
        }
        if (data?.act === 'quit') {
            message.warn(data.msg ? translate(data.msg) : i18n.t('unknownError'));
            conn = false;
            ws.close();
        }
    }

    return (
        <DraggableModal
            draggable={true}
            maskClosable={false}
            destroyOnClose={true}
            modalTitle={`${title} ${formatSize(bandwidth)}/s FPS: ${fps}`}
            footer={null}
            height={480}
            width={940}
            bodyStyle={{
                padding: 0
            }}
            {...props}
        >
            <canvas
                id='painter'
                ref={canvasRef}
                style={{width: '100%', height: '100%'}}
            />
            <Button
                style={{right:'59px'}}
                className='header-button'
                icon={<FullscreenOutlined />}
                onClick={fullScreen}
            />
            <Button
                style={{right:'115px'}}
                className='header-button'
                icon={<ReloadOutlined />}
                onClick={refresh}
            />
        </DraggableModal>
    );
}

function encrypt(data, secret) {
    let json = JSON.stringify(data);
    json = CryptoJS.enc.Utf8.parse(json);
    let encrypted = CryptoJS.AES.encrypt(json, secret, {
        mode: CryptoJS.mode.CTR,
        iv: secret,
        padding: CryptoJS.pad.NoPadding
    });
    return ws2ua(encrypted.ciphertext);
}
function decrypt(data, secret) {
    data = CryptoJS.lib.WordArray.create(data);
    let decrypted = CryptoJS.AES.encrypt(data, secret, {
        mode: CryptoJS.mode.CTR,
        iv: secret,
        padding: CryptoJS.pad.NoPadding
    });
    return ab2str(ws2ua(decrypted.ciphertext).buffer);
}

export default ScreenModal;