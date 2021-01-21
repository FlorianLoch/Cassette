const fs = require("fs")
const packageJson = fs.readFileSync('./package.json')
const version = JSON.parse(packageJson).version || 0
const webpack = require("webpack")

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
      new webpack.DefinePlugin({
        "process.env": {
          PACKAGE_VERSION: `"${version}"`,
          BUILD_DATE: new Date().toUTCString()
        }
      })
    ]
  }
}