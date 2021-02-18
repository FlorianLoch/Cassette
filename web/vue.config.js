const fs = require("fs")
const packageJson = fs.readFileSync('./package.json')
const version = JSON.parse(packageJson).version || 0
const webpack = require("webpack")
const child_process = require('child_process');

module.exports = {
  devServer: {
    port: 8083,
    host: "localhost",
    proxy: "http://localhost:8082"
  },
  pages: {
    index: {
      entry: "src/main.js",
      title: "Cassette"
    }
  },
  configureWebpack: {
    plugins: [
      new webpack.EnvironmentPlugin({
        GIT_VERSION: process.env.GIT_VERSION,
        GIT_AUTHOR_DATE: process.env.GIT_AUTHOR_DATE,
        BUILD_DATE: process.env.BUILD_DATE
      })
    ]
  }
}