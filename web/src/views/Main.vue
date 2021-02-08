<template lang="pug">
div
  .active-device-list(
    :class="{ 'no-active-devices': activeDevices.length == 0 }"
  )
    div(v-if="activeDevices.length > 0") Currently active device(s):
      li(v-for="device in activeDevices") {{ device.name }}
    div(v-else)
      button.btn.btn-primary.btn-sm(@click="fetchActiveDevices()") No active device. Click to Refresh.

  .container
    .row.mt-4
      .col-md-4(v-for="(state, slotNumber) in playerStates")
        .card.mb-4.box-shadow
          img.card-img-top(
            :src="state.albumArtLargeURL",
            alt="Album art provided by Spotify"
          )
          .card-body
            .card-content
              h5.card-title {{ state.trackName }}
              p.card-text
                i.fas.fa-compact-disc.spacer
                | {{ state.albumName }}
              p.card-text
                i.fas.fa-user-friends.spacer
                | {{ state.artistName }}
              p.card-text
                i.fas.fa-hourglass-end.spacer
                | {{ state.progress | time }} / {{ state.duration | time }}
            .row.mt-2
              .col-lg-4.p-1
                button.btn-block.btn.btn-primary(
                  @click="updatePlayerState(slotNumber)",
                  :disabled="activeDevices.length == 0"
                )
                  i.fas.fa-stop-circle
              .col-lg-4.p-1
                template(v-if="activeDevices.length > 1")
                  b-dropdown(
                    split
                    @click="restoreFromPlayerState(slotNumber)"
                    variant="success"
                  ).btn-block
                    template(#button-content)
                      i.fas.fa-play-circle
                    b-dropdown-item.disabled Start playback on:
                    b-dropdown-divider
                    b-dropdown-item(
                      v-for="device in activeDevices",
                      @click="restoreFromPlayerState(slotNumber, device.id)"
                      :key="device.name"
                    ) {{ device.name }}
                template(v-else)
                  b-button(@click="restoreFromPlayerState(slotNumber)" variant="success")
              .col-lg-4.p-1
                button.btn-block.btn.btn-danger(
                  @click="deletePlayerState(slotNumber)"
                )
                  i.fas.fa-trash-alt
      .col-md-4
        .card.mb-4.box-shadow
          button.btn.btn-block.btn-primary.store-state-btn(
            type="button",
            @click="storePlayerState",
            :disabled="activeDevices.length == 0"
          )
            i.fas.fa-stop-circle
  div
</template>

<script>
export default {
  name: "Main",
  data: function () {
    return {
      playerStates: [],
      activeDevices: [],
      showModal: false,
      errorMessage: ""
    }
  },
  filters: {
    time: function (millis) {
      const inSecs = Math.round(millis / 1000)
      const hours = Math.floor(inSecs / 3600)
      const remaining = inSecs - hours * 3600
      const minutes = Math.floor(remaining / 60)
      const seconds = remaining - minutes * 60

      return `${(hours > 0) ? hours + ":" : ""}${(minutes < 10) ? "0" : ""}${minutes}:${(seconds < 10) ? "0" : ""}${seconds}`
    }
  },
  methods: {
    showErrorMessage: function (msg) {
      this.$bvModal.msgBoxOk("Oh no! " + msg)
    },
    fetchActiveDevices: function () {
      this.$api.fetchActiveDevices().then((activeDevices) => {
        this.activeDevices = activeDevices
      }, (err) => {
        this.showErrorMessage("Failed requesting active devices.")
        console.error("Failed requesting actives devices from backend.", err)
      })
    },
    fetchPlayerStates: function () {
      this.$api.fetchPlayerStates().then((playerStates) => {
        this.playerStates = playerStates
      }, (err) => {
        this.showErrorMessage("Failed requesting your player states.")
        console.error("Failed requesting player states from backend.", err)
      })
    },
    updatePlayerState: function (slotNumber) {
      this.$api.updatePlayerState(slotNumber).then(() => {
        console.info(`Successfully updated player state in slot ${slotNumber}.`)
        this.fetchPlayerStates()
      }, (err) => {
        this.showErrorMessage("Failed updating player state.")
        console.error(`Failed updating player state in slot ${slotNumber}.`, err)
      })
    },
    storePlayerState: function () {
      this.$api.storePlayerState().then(() => {
        console.info("Successfully updated player state in new slot.")
        this.fetchPlayerStates()
      }, (err) => {
        this.showErrorMessage("Failed storing new player state.")
        console.error("Failed storing new player state.", err)
      })
    },
    deletePlayerState: function (slotNumber) {
      this.$api.deletePlayerState(slotNumber).then(() => {
        console.info(`Successfully deleted player state in slot ${slotNumber}.`)

        // Avoid re-fetching the player states, we can compute the change locally
        this.playerStates.splice(slotNumber, 1);
      }, (err) => {
        this.showErrorMessage("Failed deleting the player state.")
        console.error(`Failed deleting player state in slot ${slotNumber}.`, err)
      })
    },
    restoreFromPlayerState: function (slotNumber, deviceID) {
      this.$api.restoreFromPlayerState(slotNumber, deviceID).then(() => {
        console.info(`Successfully restored player state from slot ${slotNumber} on device ${deviceID}.`)
      }, (err) => {
        this.showErrorMessage(`Failed restoring player state on ${(deviceID !== undefined) ? `device ${deviceID}` : "currently active device"}.`)
        console.error(`Failed restoring player state from slot ${slotNumber} on device ${deviceID}.`, err)
      })
    }
  },
  mounted: function () {
    this.$api.fetchCSRFToken().then((csrfToken) => {
      console.info("Successfully fetched CSRF token.")

      this.$api.setCSRFToken(csrfToken)

      this.fetchActiveDevices()
      this.fetchPlayerStates()
    }, (err) => {
      this.showErrorMessage("Failed initializing the app. Please reload the page.")
      console.error("Failed fetching the CSRF token.", err)
    })
  }
}
</script>