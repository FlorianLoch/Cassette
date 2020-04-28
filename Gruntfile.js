module.exports = function (grunt) {
  grunt.initConfig({
      watch: {
          js: {
              files: ["./webui_src/js/**/*.js"],
              tasks: ["babel", "concat:js", "clean:babel"]
          },
          sass: {
              files: ["./webui_src/scss/**/*.scss"],
              tasks: ["sass", "concat:css", "clean:css"]
          },
          pug: {
              files: ["./webui_src/**/*.pug"],
              tasks: ["pug"]
          }
      },
      pug: {
          compile: {
              options: {
                  pretty: true
              },
              files: {
                  "./webui/index.html": "./webui_src/index.pug"
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
                  cwd: "./webui_src/js",
                  src: ["**/*.js"],
                  dest: "./webui/_build",
                  ext: ".babeled.js"
              }]
          }
      },
      sass: {
          options: {
          //     sourceMap: true
            implementation: require("node-sass")
          },
          dist: {
              files: {
                  "./webui/_build/style.css": "./webui_src/scss/style.scss"
              }
          }
      },
      concat: {
          js: {
              files: {
                  "./webui/script.js": [
                      "./node_modules/jquery/dist/jquery.slim.min.js",
                      "./node_modules/bootstrap/dist/js/bootstrap.bundle.min.js",
                      "./node_modules/nprogress/nprogress.js",
                      "./node_modules/vue/dist/vue.min.js",
                      "./node_modules/vue-resource/dist/vue-resource.min.js",
                      "./webui/_build/*.babeled.js"
                  ]
              }
          },
          css: {
              files: {
                  "./webui/style.css": [
                      "./node_modules/bootstrap/dist/css/bootstrap.min.css",
                      "./node_modules/nprogress/nprogress.css",
                      "./webui/_build/style.css"
                  ]
              }
          }
      },
    //   copy: {
    //       favicon: {
    //           src: "./webui_src/favicon.ico",
    //           dest: "./webui/favicon.ico"
    //       }
    //   },
    // Do not forget to add "copy" again to default task
      clean: {
          options: {
              force: true
          },
          babel: ["./webui/_build/*.babeled.js"],
          css: ["./webui/_build/style.css"] // this one gets merged with dependencies to style.css
      }
  });

  require("load-grunt-tasks")(grunt);

  grunt.registerTask("default", ["pug", "babel", "sass", "concat", "clean"]);
  grunt.registerTask("w", ["watch"]);
};