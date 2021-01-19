module.exports = function (grunt) {
  grunt.initConfig({
      watch: {
          js: {
              files: ["./web/js/**/*.js"],
              tasks: ["babel", "concat:js", "clean:babel"]
          },
          sass: {
              files: ["./web/scss/**/*.scss"],
              tasks: ["sass", "concat:css", "clean:css"]
          },
          pug: {
              files: ["./web/**/*.pug"],
              tasks: ["pug"]
          }
      },
      pug: {
          compile: {
              options: {
                  pretty: true,
                  data: {
                      version: require("./package.json").version,
                      buildDate: new Date().toUTCString()
                  }
              },
              files: {
                  "./web_dist/main.html": "./web/main.pug",
                  "./web_dist/consent.html": "./web/consent.pug"
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
                  cwd: "./web/js",
                  src: ["**/*.js"],
                  dest: "./web_dist/_build",
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
                  "./web_dist/_build/style.css": "./web/scss/style.scss"
              }
          }
      },
      concat: {
          js: {
              files: {
                  "./web_dist/main.js": [
                      "./node_modules/jquery/dist/jquery.slim.min.js",
                      "./node_modules/bootstrap/dist/js/bootstrap.bundle.min.js",
                      "./node_modules/nprogress/nprogress.js",
                      "./node_modules/vue/dist/vue.min.js",
                      "./node_modules/vue-resource/dist/vue-resource.min.js",
                      "./web_dist/_build/main.babeled.js"
                  ],
                  "./web_dist/consent.js": "./web_dist/_build/consent.babeled.js"
              }
          },
          css: {
              files: {
                  "./web_dist/style.css": [
                      "./node_modules/bootstrap/dist/css/bootstrap.min.css",
                      "./node_modules/nprogress/nprogress.css",
                      "./web_dist/_build/style.css"
                  ]
              }
          }
      },
      copy: {
          favicon: {
            expand: true,
            flatten: true,
            src: ["./web/favicon_different_sizes/*"],
            dest: "./web_dist/",
            filter: "isFile"
          }
      },
      clean: {
          options: {
              force: true
          },
          before: ["./web_dist"],
          after: ["./web_dist/_build"]
      }
  });

  require("load-grunt-tasks")(grunt);

  grunt.registerTask("default", ["clean:before", "pug", "babel", "sass", "concat", "copy", "clean:after"]);
  grunt.registerTask("w", ["watch"]);
};