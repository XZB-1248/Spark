import React from 'react';
import {ModalForm, ProFormCascader, ProFormDigit, ProFormGroup, ProFormText} from '@ant-design/pro-form';
import {post, request} from "../../utils/utils";
import prebuilt from '../../config/prebuilt.json';
import i18n from "../../locale/locale";

function Generate(props) {
	const initValues = getInitValues();

	async function onFinish(form) {
		if (form?.ArchOS?.length === 2) {
			form.os = form.ArchOS[0];
			form.arch = form.ArchOS[1];
			delete form.ArchOS;
		}
		form.secure = location.protocol === 'https:' ? 'true' : 'false';
		let basePath = location.origin + location.pathname + 'api/client/';
		request(basePath + 'check', form).then(res => {
			if (res.data.code === 0) {
				post(basePath += 'generate', form);
			}
		}).catch();
	}

	function getInitValues() {
		let initValues = {
			host: location.hostname,
			port: location.port,
			path: location.pathname,
			ArchOS: ['windows', 'amd64']
		};
		if (String(location.port).length === 0) {
			initValues.port = location.protocol === 'https:' ? 443 : 80;
		}
		return initValues;
	}

	return (
		<ModalForm
			modalProps={{
				destroyOnClose: true,
				maskClosable: false,
			}}
			initialValues={initValues}
			onFinish={onFinish}
			submitter={{
				render: (_, elems) => elems.pop()
			}}
			{...props}
		>
			<ProFormGroup>
				<ProFormText
					width="md"
					name="host"
					label={i18n.t('GENERATOR.HOST')}
					rules={[{
						required: true
					}]}
				/>
				<ProFormDigit
					width="md"
					name="port"
					label={i18n.t('GENERATOR.PORT')}
					min={1}
					max={65535}
					rules={[{
						required: true
					}]}
				/>
			</ProFormGroup>
			<ProFormGroup>
				<ProFormText
					width="md"
					name="path"
					label={i18n.t('GENERATOR.PATH')}
					rules={[{
						required: true
					}]}
				/>
				<ProFormCascader
					width="md"
					name="ArchOS"
					label={i18n.t('GENERATOR.OS_ARCH')}
					request={() => prebuilt}
					rules={[{
						required: true
					}]}
				/>
			</ProFormGroup>
		</ModalForm>
	)
}

export default Generate;