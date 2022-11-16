import i18n from 'i18next';
import zhCN from './zh-CN';
import en from './en';

const resources = {
	'en': {
		translation: en
	},
	'en-US': {
		translation: en
	},
	'zh-CN': {
		translation: zhCN
	},
};
const lang = navigator.language && navigator.language.length ? navigator.language : 'en';
const locale = resources[lang] ? lang : 'en';

i18n.init({
	lng: lang,
	fallbackLng: 'en',
	initImmediate: true,
	resources
});

function getLang() {
	return lang;
}
function getLocale() {
	return locale;
}

export { getLang, getLocale };
export default i18n;