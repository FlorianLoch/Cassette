import axios from "axios"

const CSRF_HEADER_NAME = "X-Cassette-CSRF".toLowerCase()
const API_PATH = "/api"
const URL_DATA = API_PATH + "/you"
const URL_CSRF_TOKEN = API_PATH + "/csrfToken"
const URL_PLAYER_STATES = API_PATH + "/playerStates"
const URL_ACTIVE_DEVICES = API_PATH + "/activeDevices"
const CONSENT_COOKIE_NAME = "cassette_consent"

const API = function (options) {
    const client = options ? options.axios : null || axios.create()

    this.fetchCSRFToken = () => {
        return client.head(URL_CSRF_TOKEN).then((res) => {
            return res.headers[CSRF_HEADER_NAME]
        })
    }

    this.setCSRFToken = (csrfToken) => {
        client.defaults.headers.common[CSRF_HEADER_NAME] = csrfToken
    }

    this.fetchActiveDevices = () => {
        return client.get(URL_ACTIVE_DEVICES).then((res) => {
            return res.data
        })
    }

    this.fetchPlayerStates = () => {
        return client.get(URL_PLAYER_STATES).then((res) => {
            return preparePlayerStates(res.data)
        })
    }

    // Sorts the states by LRU and adds the original index as slotNumber
    function preparePlayerStates(rawPlayerStates) {
        const cookedPlayerStates = rawPlayerStates.map((cur, idx) => {
            return {
                state: cur,
                slotNumber: idx,
            }
        })

        cookedPlayerStates.sort((a, b) => {
            return b.state.suspendedAtTs - a.state.suspendedAtTs
        })

        return cookedPlayerStates
    }

    this.updatePlayerState = (slotNumber) => {
        return client.put(`${URL_PLAYER_STATES}/${slotNumber}`)
    }

    this.storePlayerState = () => {
        return client.post(URL_PLAYER_STATES)
    }

    this.deletePlayerState = (slotNumber) => {
        return client.delete(`${URL_PLAYER_STATES}/${slotNumber}`)
    }

    this.restoreFromPlayerState = (slotNumber, deviceID) => {
        const url = `${URL_PLAYER_STATES}/${slotNumber}/restore${
            deviceID ? `?deviceID=${deviceID}` : ""
        }`
        return client.post(url)
    }

    this.deleteYourData = () => {
        return client.delete(URL_DATA)
    }

    this.giveConsent = API.giveConsent

    this.withdrawConsent = API.withdrawConsent

    this.consentGiven = API.consentGiven

    this.URL_DATA = URL_DATA
}

API.install = function (Vue, options) {
    Vue.prototype.$api = new API(options)
}

API.giveConsent = () => {
    const now = Math.floor(Date.now() / 1000)
    const maxAge = 10 * 60 * 60 * 24 * 365 // 10 years, keep this in sync with consent middleware in the backend

    document.cookie = `${CONSENT_COOKIE_NAME}=${now};max-age=${maxAge}`
}

API.withdrawConsent = () => {
    document.cookie = `${CONSENT_COOKIE_NAME}="consent withdrawn"`
}

API.consentGiven = () => {
    return document.cookie.split(";").some((it) => {
        return (
            it.trim().startsWith(CONSENT_COOKIE_NAME + "=") &&
            it.indexOf("withdrawn") < 0
        )
    })
}

export default API
