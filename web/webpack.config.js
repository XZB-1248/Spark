const path = require("path");
const esbuild = require('esbuild');
const HtmlWebpackPlugin = require("html-webpack-plugin");
const CopyWebpackPlugin = require("copy-webpack-plugin");
const {ESBuildMinifyPlugin} = require("esbuild-loader");
const {CleanWebpackPlugin} = require("clean-webpack-plugin");
const AntdDayjsWebpackPlugin = require("antd-dayjs-webpack-plugin");
const CompressionPlugin = require("compression-webpack-plugin");

module.exports = (env, args) => {
    let mode = args.mode;
    return {
        entry: path.join(__dirname, 'src/index.jsx'),
        output: {
            publicPath: mode === 'development' ? undefined : './',
            path: path.resolve(__dirname, 'dist'),
            filename: '[name].[contenthash:7].js'
        },
        devtool: mode === 'development' ? 'eval-source-map' : false,
        module: {
            rules: [
                {
                    test: /\.(js|jsx)$/i,
                    loader: 'esbuild-loader',
                    include: path.resolve(__dirname, 'src'),
                    options: {
                        loader: 'jsx',
                        target: 'es2015'
                    }
                },
                {
                    test: /\.css$/,
                    use: [
                        'style-loader',
                        'css-loader',
                        {
                            loader: 'esbuild-loader',
                            options: {
                                loader: 'css',
                                minify: true
                            }
                        }
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
                threshold: 128 * 1024,
                compressionOptions: {
                    level: 9
                }
            })
        ],
        optimization: {
            minimize: mode === 'production',
            minimizer: [
                new ESBuildMinifyPlugin({
                    css: true,
                    target: 'es2015',
                    implementation: esbuild,
                    legalComments: 'none'
                })
            ],
            runtimeChunk: 'single',
            splitChunks: mode === 'development' ? false : {
                chunks: 'all',
                filename: '[name].chunk.[contenthash:7].js',
                cacheGroups: {
                    react: {
                        test: /react|redux|react-router/i,
                        priority: -1,
                        reuseExistingChunk: true
                    },
                    common: {
                        test: /axios|i18next|dayjs/i,
                        priority: -2,
                        reuseExistingChunk: true
                    },
                    proForm: {
                        test: /pro-form/i,
                        priority: -3,
                        reuseExistingChunk: true
                    },
                    proTable: {
                        test: /pro-table/i,
                        priority: -3,
                        reuseExistingChunk: true
                    },
                    proLayout: {
                        test: /pro-layout/i,
                        priority: -3,
                        reuseExistingChunk: true
                    },
                    antd: {
                        test: /antd|ant-design/i,
                        priority: -4,
                        reuseExistingChunk: true
                    },
                    ace: {
                        test: /react-ace|ace-builds/i,
                        priority: -5,
                        reuseExistingChunk: true
                    },
                    xterm: {
                        test: /xterm/i,
                        priority: -6,
                        reuseExistingChunk: true
                    },
                    vendor: {
                        test: /[\\/]node_modules[\\/]/i,
                        priority: -7,
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