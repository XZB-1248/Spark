import React from "react";
import {Spin} from "antd";
import {LoadingOutlined} from "@ant-design/icons";

function Suspense(props) {
	return (
		<React.Suspense fallback={<Spin indicator={<LoadingOutlined />} />}>
			{props.children}
		</React.Suspense>
	)
}

export default Suspense;