(() => {
  document.getElementById("consent-btn").addEventListener("click", () => {
    const now = encodeURIComponent(new Date().toUTCString())
    const maxAge = 10*60*60*24*365 // 10 years
    document.cookie = `cassette_consent=${now};max-age=${maxAge}`

    window.location.href = "/"
  })
})()