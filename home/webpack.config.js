const webpack = require('webpack');
const path = require('path');
const CopyPlugin = require('copy-webpack-plugin');

const env = process.env.NODE_ENV || 'development';

const config = {
    mode: env,
    devtool: env === 'development' ? 'inline-source-map' : false,
    entry: {
        login: './cp/login.tsx',
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
        new CopyPlugin({
            patterns: [
                {from: 'cp/favicon.ico', to: path.resolve(__dirname, 'gen')}
            ]
        })
    ]
};

if (process.env.USE_BUNDLE_ANALYZER) {
    const BundleAnalyzerPlugin = require('webpack-bundle-analyzer').BundleAnalyzerPlugin;

    config.plugins.push(
        new BundleAnalyzerPlugin({
            openAnalyzer: false
        })
    );
}

module.exports = config;
