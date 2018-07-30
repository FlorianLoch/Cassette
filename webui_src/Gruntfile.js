module.exports = function (grunt) {
  grunt.initConfig({
      watch: {
          js: {
              files: ["./js/**/*.js"],
              tasks: ["babel", "concat:js", "clean:babel"]
          },
          sass: {
              files: ["./scss/**/*.scss"],
              tasks: ["sass", "concat:css", "clean:css"]
          },
          pug: {
              files: ["./**/*.pug"],
              tasks: ["pug"]
          }
      },
      pug: {
          compile: {
              options: {
                  pretty: true
              },
              files: {
                  "../webui/index.html": "./index.pug"
              }
          }
      },
      babel: {
          options: {
              sourceMap: false,
              presets: ["es2015"]
          },
          dist: {
              files: [{
                  expand: true,
                  cwd: "./js",
                  src: ["**/*.js"],
                  dest: "../webui/_build",
                  ext: ".babeled.js"
              }]
          }
      },
      sass: {
          // options: {
          //     sourceMap: true
          // },
          dist: {
              files: {
                  "./webui/_build/style.css": "./scss/*.scss"
              }
          }
      },
      concat: {
          js: {
              files: {
                  "../webui/script.js": [
                      "node_modules/nprogress/nprogress.js",
                      "node_modules/vue/dist/vue.min.js",
                      "node_modules/vue-resource/dist/vue-resource.min.js",
                      "../webui/_build/*.babeled.js"
                  ]
              }
          },
          css: {
              files: {
                  "../webui/style.css": [
                      "node_modules/bootstrap/dist/css/bootstrap.min.css",
                      "node_modules/nprogress/nprogress.css",
                      "../webui/_build/style.css"
                  ]
              }
          }
      },
      copy: {
          favicon: {
              src: "./favicon.ico",
              dest: "./dist/favicon.ico"
          }
      },
      clean: {
          options: {
              force: true
          },
          babel: ["../webui/_build/*.babeled.js"],
          css: ["../webui/_build/style.css"] // this one gets merged with dependencies to style.css
      }
  });

  require("load-grunt-tasks")(grunt);

  grunt.registerTask("default", ["pug", "babel", "sass", "copy", "concat", "clean"]);
  grunt.registerTask("w", ["watch"]);
};