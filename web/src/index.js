import React from 'react';
import ReactDOM from 'react-dom';
import {HashRouter as Router, Route, Routes} from 'react-router-dom';
import Wrapper from './components/wrapper';
import Err from './pages/404';
import axios from 'axios';
import {message} from 'antd';
import i18n from "./locale/locale";

import './global.css';
import 'antd/dist/antd.css';
import Overview from "./pages/overview";
import {translate} from "./utils/utils";

axios.defaults.baseURL = '.';
axios.interceptors.response.use(async res => {
    let data = res.data;
    if (data.hasOwnProperty('code')) {
        if (data.code !== 0){
            message.warn(translate(data.msg));
        } else {
            // The first request will ask user to provide user/pass.
            // If set timeout at the beginning, then timeout warning
            // might be triggered before authentication finished.
            axios.defaults.timeout = 5000;
        }
    }
    return Promise.resolve(res);
}, err => {
    // console.error(err);
    if (err.code === 'ECONNABORTED') {
        message.error(i18n.t('COMMON.REQUEST_TIMEOUT'));
        return Promise.reject(err);
    }
    let res = err.response;
    let data = res?.data ?? {};
    if (data.hasOwnProperty('code') && data.hasOwnProperty('msg')) {
        if (data.code !== 0){
            message.warn(translate(data.msg));
            return Promise.resolve(res);
        }
    }
    return Promise.reject(err);
});

ReactDOM.render(
    <Router>
        <Routes>
            <Route path="/" element={<Wrapper><Overview/></Wrapper>}/>
            <Route path="*" element={<Err/>}/>
        </Routes>
    </Router>,
    document.getElementById('root')
);