const path = require('path');
const TerserPlugin = require('terser-webpack-plugin');
const HtmlWebpackPlugin = require('html-webpack-plugin');
const UglifyJsPlugin = require('uglifyjs-webpack-plugin');
const {CleanWebpackPlugin} = require('clean-webpack-plugin');
const AntdDayjsWebpackPlugin = require('antd-dayjs-webpack-plugin');

module.exports = (env, args) => {
    let mode = args.mode;
    return {
        entry: './src/index.js',
        output: {
            publicPath: mode === 'development' ? undefined : './',
            path: path.resolve(__dirname, 'dist'),
            filename: '[name].js' //'[name].[contenthash:7].js'
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
                                    modifyVars: {'@primary-color': '#1DA57A'},
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
        plugins: mode === 'production' ? [
            new HtmlWebpackPlugin({
                appMountId: 'root',
                template: './public/index.html',
                filename: 'index.html',
                inject: true
            }),
            new CleanWebpackPlugin(),
            new AntdDayjsWebpackPlugin()
        ] : [
            new HtmlWebpackPlugin({
                appMountId: 'root',
                template: './public/index.html',
                filename: 'index.html',
                inject: true
            }),
            new CleanWebpackPlugin(),
            new AntdDayjsWebpackPlugin()
        ],
        optimization: {
            minimize: mode === 'production',
            minimizer: [
                new TerserPlugin({
                    extractComments: false,
                    terserOptions: {
                        compress: {
                            drop_console: mode === 'production'
                        }
                    }
                }),
                new UglifyJsPlugin({
                    test: /\.js(\?.*)?$/i,
                    chunkFilter: (chunk) => chunk.name !== 'vendor',
                    cache: true,
                    parallel: 5,
                    sourceMap: mode === 'development',
                    uglifyOptions: {
                        compress: {
                            drop_console: mode === 'production',
                            collapse_vars: true,
                            reduce_vars: true,
                        },
                        output: {
                            beautify: mode === 'production',
                            comments: mode === 'development',
                        }
                    }
                })
            ],
            runtimeChunk: 'single',
            splitChunks: {
                chunks: 'initial',
                cacheGroups: {
                    runtime: {
                        name: 'runtime',
                        test: (module) => {
                            return /axios|react|redux|antd|ant-design/.test(module.context);
                        },
                        chunks: 'initial',
                        priority: 10,
                        reuseExistingChunk: true
                    },
                    vendor: {
                        test: /[\\/]node_modules[\\/]/,
                        name: 'vendors',
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
                    target: 'https://1248.ink/spark/',
                    secure: false
                }
            }
        }
    };
};