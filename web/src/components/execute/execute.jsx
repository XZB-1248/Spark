import React from 'react';
import {ModalForm, ProFormText} from '@ant-design/pro-form';
import {request} from "../../utils/utils";
import i18n from "../../locale/locale";
import {message} from "antd";

function Execute(props) {
	async function onFinish(form) {
		form.device = props.device.id;
		let basePath = location.origin + location.pathname + 'api/device/';
		request(basePath + 'exec', form).then(res => {
			if (res.data.code === 0) {
				message.success(i18n.t('RUNNER.EXECUTION_SUCCESS'));
			}
		});
	}

	return (
		<ModalForm
			modalProps={{
				destroyOnClose: true,
				maskClosable: false,
			}}
			title={i18n.t('RUNNER.TITLE')}
			width={380}
			onFinish={onFinish}
			onVisibleChange={open => {
				if (!open) props.onCancel();
			}}
			submitter={{
				render: (_, elems) => elems.pop()
			}}
			{...props}
		>
			<ProFormText
				width="md"
				name="cmd"
				label={i18n.t('RUNNER.CMD_PLACEHOLDER')}
				rules={[{
					required: true
				}]}
			/>
			<ProFormText
				width="md"
				name="args"
				label={i18n.t('RUNNER.ARGS_PLACEHOLDER')}
			/>
		</ModalForm>
	)
}

export default Execute;