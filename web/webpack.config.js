const path = require("path");
const TerserPlugin = require("terser-webpack-plugin");
const HtmlWebpackPlugin = require("html-webpack-plugin");
const CopyWebpackPlugin = require("copy-webpack-plugin");
const {CleanWebpackPlugin} = require("clean-webpack-plugin");
const AntdDayjsWebpackPlugin = require("antd-dayjs-webpack-plugin");
const CompressionPlugin = require("compression-webpack-plugin");

module.exports = (env, args) => {
    let mode = args.mode;
    return {
        entry: path.join(__dirname, 'src/index.js'),
        output: {
            publicPath: mode === 'development' ? undefined : './',
            path: path.resolve(__dirname, 'dist'),
            filename: '[name].[contenthash:7].js'
        },
        devtool: mode === 'development' ? 'eval-source-map' : false,
        module: {
            rules: [
                {
                    test: /\.(js|jsx)$/,
                    use: 'babel-loader',
                    exclude: /node_modules/
                },
                {
                    test: /\.css$/,
                    use: [
                        'style-loader',
                        'css-loader'
                    ]
                },
                {
                    test: /\.less$/,
                    use: [
                        'style-loader',
                        'css-loader',
                        {
                            loader: 'less-loader',
                            options: {
                                lessOptions: {
                                    javascriptEnabled: true
                                }
                            }
                        }
                    ]
                }
            ]
        },
        resolve: {
            extensions: [
                '.js',
                '.jsx'
            ]
        },
        plugins: [
            new HtmlWebpackPlugin({
                appMountId: 'root',
                template:  path.resolve(__dirname, 'public/index.html'),
                filename: 'index.html',
                inject: true
            }),
            new CleanWebpackPlugin(),
            new AntdDayjsWebpackPlugin(),
            new CopyWebpackPlugin({
                patterns: [
                    {
                        from: path.resolve(__dirname, 'public/ace.js'),
                    },
                    {
                        from: path.resolve(__dirname, 'public/ext-modelist.js'),
                    }
                ]
            }),
            new CompressionPlugin({
                test: /\.js$|\.css$|\.html$/,
                filename: "[file].gz",
                algorithm: "gzip",
                threshold: 256 * 1024,
                compressionOptions: {
                    level: 9
                }
            })
        ],
        optimization: {
            minimize: mode === 'production',
            minimizer: [
                new TerserPlugin({
                    test: /\.js(\?.*)?$/i,
                    parallel: true,
                    extractComments: false,
                    terserOptions: {
                        compress: {
                            drop_console: false,
                            collapse_vars: true,
                            reduce_vars: true,
                        }
                    }
                })
            ],
            runtimeChunk: 'multiple',
            splitChunks: {
                chunks: 'initial',
                cacheGroups: {
                    react: {
                        test: /react|redux|react-router/i,
                        priority: -1,
                        chunks: 'all',
                        reuseExistingChunk: true
                    },
                    common: {
                        test: /axios|i18next|crypto-js|dayjs/i,
                        priority: -2,
                        chunks: 'all',
                        reuseExistingChunk: true
                    },
                    antd: {
                        test: /antd|ant-design/i,
                        priority: -3,
                        chunks: 'all',
                        reuseExistingChunk: true
                    },
                    addon: {
                        test: /xterm|react-ace|ace-builds/i,
                        priority: -4,
                        chunks: 'initial',
                        reuseExistingChunk: true
                    },
                    vendor: {
                        test: /[\\/]node_modules[\\/]/i,
                        priority: -5,
                        chunks: 'initial',
                        reuseExistingChunk: true
                    }
                }
            }
        },
        devServer: {
            port: 3000,
            open: true,
            hot: true,
            proxy: {
                '/api/': {
                    target: 'http://localhost:8001/',
                    secure: false
                },
                '/api/device/desktop': {
                    target: 'ws://localhost:8001/',
                    ws: true
                },
                '/api/device/terminal': {
                    target: 'ws://localhost:8001/',
                    ws: true
                },
            }
        }
    };
};