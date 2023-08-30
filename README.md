# Log Data Collection and Visualization :bar_chart:

This project aims to collect log data from the `/var/log/auth.log` file, extract IP addresses and usernames from failed login attempts, and use API calls to get geographical data such as longitude, latitude, city, country, and more. The collected data is then recorded as Prometheus metrics and can be visualized using Grafana.

## Prerequisites :gear:

- Go (Golang) programming language
- Prometheus
- Grafana
- `github.com/fsnotify/fsnotify` library
- `github.com/prometheus/client_golang/prometheus` library
- `github.com/prometheus/client_golang/prometheus/promhttp` library

## Setup :wrench:

1. Clone the repository to your local machine:

   ```bash
   git clone https://github.com/whosFritz/geo-mapping-ivan.git
   cd geo-mapping-ivan
   ```

2. Install the required Go libraries:
   ```bash
   go get -u github.com/fsnotify/fsnotify
   go get -u github.com/prometheus/client_golang/prometheus
   go get -u github.com/prometheus/client_golang/prometheus/promhttp
   ```
3. Create a .env file in the project directory and add your API token:

   - u can get an api token from [here](https://www.findip.net/)
   - TOKEN=your_api_token_here

4. Build the project:

   ```bash
   go build
   ```

## Usage :computer:

1. Start Prometheus
   - Prometheus will start and collect metrics exposed by the application.
2. Start the application:

   ```bash
   ./geo-mapping-van
   ```

   The application will monitor changes to the log.txt file and extract data from failed login attempts. It will then record the data as Prometheus metrics.

3. Start Grafana and set up Prometheus as a data source.

4. Import the provided Grafana dashboard to visualize the recorded metrics.


## Grafana Dashboard :chart_with_upwards_trend:
A sample Grafana dashboard JSON file is provided in the grafana_dashboard.json file. You can import this dashboard into Grafana to visualize the recorded metrics. The dashboard will display information about failed login attempts, including IP addresses, usernames, geographical data, and more.

## Customize :hammer_and_wrench:
Feel free to customize the code and the Grafana dashboard to fit your needs. You can modify the Prometheus metrics, add more labels, or create new visualizations in Grafana to gain insights from the collected data.

## Note :memo:
Make sure to keep your API token secure by storing it in the .env file and adding that file to your .gitignore to prevent accidentally sharing your sensitive information.

## License :scroll:
This project is licensed under the MIT License. [MIT-License](LICENSE)