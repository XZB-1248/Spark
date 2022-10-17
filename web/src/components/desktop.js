import React, {useCallback, useEffect, useState} from 'react';
import {encrypt, decrypt, formatSize, genRandHex, getBaseURL, translate} from "../utils/utils";
import i18n from "../locale/locale";
import DraggableModal from "./modal";
import CryptoJS from "crypto-js";
import {Button, message} from "antd";
import {FullscreenOutlined, ReloadOutlined} from "@ant-design/icons";

let ws = null;
let ctx = null;
let conn = false;
let canvas = null;
let secret = null;
let ticker = 0;
let frames = 0;
let bytes = 0;
let ticks = 0;
let title = i18n.t('DESKTOP.TITLE');
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
        ctx = canvas.getContext('2d', {alpha: false});
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
                    message.warn(i18n.t('COMMON.DISCONNECTED'));
                }
            };
            ws.onerror = (e) => {
                console.error(e);
                if (conn) {
                    conn = false;
                    message.warn(i18n.t('COMMON.DISCONNECTED'));
                } else {
                    message.warn(i18n.t('COMMON.CONNECTION_FAILED'));
                }
            };
            clearInterval(ticker);
            ticker = setInterval(() => {
                setBandwidth(bytes);
                setFps(frames);
                bytes = 0;
                frames = 0;
                ticks++;
                if (ticks > 10 && conn) {
                    ticks = 0;
                    ws.send(encrypt({
                        act: 'DESKTOP_PING'
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
                    act: 'DESKTOP_SHOT'
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
            let width = dv.getUint16(1, false);
            let height = dv.getUint16(3, false);
            if (width === 0 || height === 0) return;
            canvas.width = width;
            canvas.height = height;
            return;
        }
        if (op === 0) frames++;
        bytes += ab.byteLength;
        let offset = 1;
        while (offset < ab.byteLength) {
            let it = dv.getUint16(offset + 0, false); // image type
            let il = dv.getUint16(offset + 2, false); // image length
            let dx = dv.getUint16(offset + 4, false); // image block x
            let dy = dv.getUint16(offset + 6, false); // image block y
            let bw = dv.getUint16(offset + 8, false); // image block width
            let bh = dv.getUint16(offset + 10, false); // image block height
            offset += 12;
            updateImage(ab.slice(offset, offset + il), it, dx, dy, bw, bh, canvasCtx);
            offset += il;
        }
        dv = null;
    }
    function updateImage(ab, it, dx, dy, bw, bh, canvasCtx) {
        if (it === 0) {
            canvasCtx.putImageData(new ImageData(new Uint8ClampedArray(ab), bw, bh), dx, dy, 0, 0, bw, bh);
        } else {
            createImageBitmap(new Blob([ab]), 0, 0, bw, bh)
                .then((ib) => {
                    canvasCtx.drawImage(ib, 0, 0, bw, bh, dx, dy, bw, bh);
                });
        }
    }
    function handleJSON(ab) {
        let data = decrypt(ab, secret);
        try {
            data = JSON.parse(data);
        } catch (_) {}
        if (data?.act === 'WARN') {
            message.warn(data.msg ? translate(data.msg) : i18n.t('COMMON.UNKNOWN_ERROR'));
            return;
        }
        if (data?.act === 'QUIT') {
            message.warn(data.msg ? translate(data.msg) : i18n.t('COMMON.UNKNOWN_ERROR'));
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

export default ScreenModal;