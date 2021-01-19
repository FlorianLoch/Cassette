(() => {
  document.getElementById("consent-btn").addEventListener("click", () => {
    document.cookie = `cassette_consent=${encodeURIComponent(new Date().toUTCString())}`

    window.location.href = "/"
  })
})()