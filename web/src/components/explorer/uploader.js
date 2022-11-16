import React, {useEffect, useState} from "react";
import Qs from "qs";
import {formatSize, preventClose} from "../../utils/utils";
import axios from "axios";
import {message, Modal, Progress, Typography} from "antd";
import i18n from "../../locale/locale";
import DraggableModal from "../modal";

let abortController = null;
function FileUploader(props) {
	const [open, setOpen] = useState(!!props.file);
	const [percent, setPercent] = useState(0);
	const [status, setStatus] = useState(0);
	// 0: ready, 1: uploading, 2: success, 3: fail, 4: cancel

	useEffect(() => {
		setStatus(0);
		if (props.file) {
			setOpen(true);
			setPercent(0);
		}
	}, [props.file]);

	function onConfirm() {
		if (status !== 0) {
			onCancel();
			return;
		}
		const params = Qs.stringify({
			device: props.device,
			path: props.path,
			file: props.file.name
		});
		let uploadStatus = 1;
		setStatus(1);
		window.onbeforeunload = preventClose;
		abortController = new AbortController();
		axios.post(
			'/api/device/file/upload?' + params,
			props.file,
			{
				headers: {
					'Content-Type': 'application/octet-stream'
				},
				timeout: 0,
				onUploadProgress: (progressEvent) => {
					let percentCompleted = Math.round((progressEvent.loaded * 100) / progressEvent.total);
					setPercent(percentCompleted);
				},
				signal: abortController.signal
			}
		).then(res => {
			let data = res.data;
			if (data.code === 0) {
				uploadStatus = 2;
				setStatus(2);
				message.success(i18n.t('EXPLORER.UPLOAD_SUCCESS'));
			} else {
				uploadStatus = 3;
				setStatus(3);
			}
		}).catch(err => {
			if (axios.isCancel(err)) {
				uploadStatus = 4;
				setStatus(4);
				message.error(i18n.t('EXPLORER.UPLOAD_ABORTED'));
			} else {
				uploadStatus = 3;
				setStatus(3);
				message.error(i18n.t('EXPLORER.UPLOAD_FAILED') + i18n.t('COMMON.COLON') + err.message);
			}
		}).finally(() => {
			abortController = null;
			window.onbeforeunload = null;
			setTimeout(() => {
				setOpen(false);
				if (uploadStatus === 2) {
					props.onSuccess();
				} else {
					props.onCancel();
				}
			}, 1500);
		});
	}
	function onCancel() {
		if (status === 0) {
			setOpen(false);
			setTimeout(props.onCancel, 300);
			return;
		}
		if (status === 1) {
			Modal.confirm({
				autoFocusButton: 'cancel',
				content: i18n.t('EXPLORER.UPLOAD_CANCEL_CONFIRM'),
				onOk: () => {
					abortController.abort();
				},
				okButtonProps: {
					danger: true,
				},
			});
			return;
		}
		setTimeout(() => {
			setOpen(false);
			setTimeout(props.onCancel, 300);
		}, 1500);
	}

	function getDescription() {
		switch (status) {
			case 1:
				return percent + '%';
			case 2:
				return i18n.t('EXPLORER.UPLOAD_SUCCESS');
			case 3:
				return i18n.t('EXPLORER.UPLOAD_FAILED');
			case 4:
				return i18n.t('EXPLORER.UPLOAD_ABORTED');
			default:
				return i18n.t('EXPLORER.UPLOAD');
		}
	}

	return (
		<DraggableModal
			centered
			draggable
			open={open}
			closable={false}
			keyboard={false}
			maskClosable={false}
			destroyOnClose={true}
			confirmLoading={status === 1}
			okText={i18n.t(status === 1 ? 'EXPLORER.UPLOADING' : 'EXPLORER.UPLOAD')}
			modalTitle={i18n.t(status === 1 ? 'EXPLORER.UPLOADING' : 'EXPLORER.UPLOAD')}
			okButtonProps={{disabled: status !== 0}}
			cancelButtonProps={{disabled: status > 1}}
			onCancel={onCancel}
			onOk={onConfirm}
			width={550}
		>
			<div>
                <span
					style={{
						whiteSpace: 'nowrap',
						fontSize: '20px',
						marginRight: '10px',
					}}
				>
                    {getDescription()}
                </span>
				<Typography.Text
					ellipsis={{rows: 1}}
					style={{maxWidth: 'calc(100% - 140px)'}}
				>
					{props.file.name}
				</Typography.Text>
				<span
					style={{whiteSpace: 'nowrap'}}
				>
					{'（'+formatSize(props.file.size)+'）'}
				</span>
			</div>
			<Progress
				strokeLinecap='butt'
				percent={percent}
				showInfo={false}
			/>
		</DraggableModal>
	)
}

export default FileUploader;