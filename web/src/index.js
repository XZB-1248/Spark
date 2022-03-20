import React from 'react';
import ReactDOM from 'react-dom';
import {HashRouter as Router, Route, Routes} from 'react-router-dom';
import Wrapper from './components/wrapper';
import Err from './pages/404';
import axios from 'axios';
import {message} from 'antd';
import dayjs from 'dayjs';

import './global.css';
import 'antd/dist/antd.css';
import 'dayjs/locale/zh-cn';
import Overview from "./pages/overview";

dayjs.locale('zh-cn');

axios.defaults.baseURL = '.';
axios.interceptors.response.use(async (res) => {
    let data = res.data;
    if (data.hasOwnProperty('code')) {
        if (data.code !== 0){
            message.warn(data.msg);
        } else {
            // The first request will ask user to provide user/pass.
            // If set timeout at the beginning, then timeout warning
            // might be triggered before authentication finished.
            axios.defaults.timeout = 5000;
        }
    }
    return Promise.resolve(res);
}, (err) => {
    if (err.code === 'ECONNABORTED') {
        message.warn('请求超时');
        return Promise.resolve(err);
    }
    let res = err.response;
    let data = res.data;
    if (data.hasOwnProperty('code')) {
        if (data.code !== 0){
            message.warn(data.msg);
        }
    }
    return Promise.resolve(res);
});

ReactDOM.render(
    <Router>
        <Routes>
            <Route path="/" element={<Wrapper><Overview/></Wrapper>}/>
            <Route
                path="*"
                element={<Err/>}
            />
        </Routes>
    </Router>,
    document.getElementById('root')
);