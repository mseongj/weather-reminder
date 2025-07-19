# weather-reminder
## my little project for learn golang

### startup background
- I often find myself wishing there was a display at the front door that could show the weather. It was this thought that inspired me to start this project.

### Goals and Skills
- My goal was to design an API as crisp as Kendrick Lamar's 'Not like Us' and animations as smooth as those like Luther.
- To achieve this, I mainly used Go and htmx.

핵심 고려사항: Orange Pi Zero 3의 성능


   1. RAM 용량 (가장 중요):
       * Orange Pi Zero 3는 1GB, 1.5GB, 2GB, 4GB 등 다양한 RAM 옵션으로 출시됩니다.
       * **최적화 후:** Go 단일 서버로 변경하여 RAM 사용량이 크게 줄었지만, 데스크탑 환경과 웹 브라우저를 함께 사용하려면 여전히 최소 1GB 이상의 RAM을 권장합니다. 2GB 모델이라면 매우 쾌적하게 사용할 수 있습니다.


   2. 운영체제(OS) 선택:
       * Orange Pi Zero 3에는 공식적으로 제공되는 Debian, Ubuntu 외에도 Armbian과 같은 가벼운 서드파티 OS를 설치할 수 있습니다.
       * 리소스 사용을 최소화하기 위해, GNOME이나 KDE 같은 무거운 데스크탑 환경 대신 XFCE나 LXDE 같은 가벼운 데스크탑 환경이 포함된 OS 이미지를 선택하는 것이 좋습니다.

  ---

  Orange Pi Zero 3 구현 방법 (1GB 이상 RAM 모델 기준)


  전체적인 과정은 Orange Pi 4와 거의 동일합니다. 가장 중요한 것은 `systemd` 서비스 파일에 실제 OS의 사용자 이름과 경로를 정확하게 입력하는 것입니다.

  1단계: OS 준비

   * Orange Pi Zero 3용 Debian 또는 Ubuntu (가벼운 데스크탑 버전 권장) 이미지를 MicroSD 카드에 설치합니다.

  2단계: 기본 소프트웨어 설치

   * 터미널을 열고 다음 명령어를 실행합니다.

   ```bash
   sudo apt update && sudo apt upgrade -y
   sudo apt install -y golang chromium-browser
   ```

  3단계: 프로젝트 파일 배포 및 빌드

   * Git으로 소스코드를 가져오고, Go 애플리케이션을 빌드합니다.

   ```bash
   git clone https://github.com/your-github-id/weather-reminder.git
   cd weather-reminder
   go build -o weather-reminder .
   ```
   * `go build` 명령어는 현재 디렉토리의 소스코드를 컴파일하여 `weather-reminder`라는 이름의 실행 파일을 생성합니다.

  4단계: 자동 시작 설정 (Systemd 및 Autostart)


   * 가장 중요한 단계입니다. Orange Pi Zero 3의 OS에 설정된 실제 사용자 이름과 홈 디렉토리 경로를 정확하게 확인하고 수정해야 합니다. (예: orangepi, root 등)


   1. 백엔드 서비스 생성 (`weather-app.service`)

      ```bash
      sudo nano /etc/systemd/system/weather-app.service
      ```

      User와 WorkingDirectory를 실제 환경에 맞게 수정하여 붙여넣습니다. `ExecStart`는 방금 빌드한 실행 파일을 가리킵니다.

      ```ini
      [Unit]
      Description=Weather Reminder Go App
      After=network.target

      [Service]
      Type=simple
      User=orangepi  # <-- 실제 사용자 이름으로 수정!
      WorkingDirectory=/home/orangepi/weather-reminder  # <-- 실제 경로로 수정!
      ExecStart=/home/orangepi/weather-reminder/weather-reminder
      Restart=on-failure

      [Install]
      WantedBy=multi-user.target
      ```

   2. 웹 브라우저 자동 실행 (키오스크 모드)
      이 과정은 이전과 동일하지만, URL을 Go 서버 포트인 8080으로 변경합니다.

      ```bash
      mkdir -p ~/.config/autostart
      nano ~/.config/autostart/weather-display.desktop
      ```

      아래 내용을 붙여넣습니다.

      ```ini
      [Desktop Entry]
      Type=Application
      Name=WeatherDisplay
      Exec=chromium-browser --kiosk http://localhost:8080
      ```

  5단계: 서비스 활성화 및 재부팅

   * 서비스를 활성화하고 재부팅합니다.

   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable weather-app.service
   sudo reboot
   ```

  결론


   * **실행 가능성:** 예, 가능합니다. Node.js를 제거하고 Go 단일 서버로 변경하여 1GB RAM 모델에서도 충분히 실행 가능합니다.
   * **핵심 설정:** systemd 서비스 파일의 `User`, `WorkingDirectory`, `ExecStart` 경로를 실제 환경에 맞게 정확히 설정하는 것이 중요합니다.
