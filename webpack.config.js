const webpack = require('webpack');
const path = require('path');
const BundleAnalyzerPlugin = require('webpack-bundle-analyzer').BundleAnalyzerPlugin;

const env = process.env.NODE_ENV || 'development';

module.exports = {
    mode: env,
    devtool: env === 'development' ? 'inline-source-map' : false,
    entry: {
        login: './cp/login.ts',
        control: './cp/control.tsx'
    },
    resolve: {
        modules: ['node_modules'],
        extensions: ['.mjs', '.ts', '.tsx', '.js']
    },
    output: {
        path: path.resolve(__dirname, 'gen'),
        filename: '[name].js',
        clean: true
    },
    module: {
        rules: [
            {
                test: /\.tsx?$/,
                exclude: /node_modules/,
                use: 'ts-loader'
            },
            {
                test: /\.scss$/,
                use: ['style-loader', 'css-loader', 'sass-loader']
            }
        ]
    },
    plugins: [
        new BundleAnalyzerPlugin({
            openAnalyzer: false
        })
    ]
};
