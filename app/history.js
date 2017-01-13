var chartOptions = {
	chartArea: {
		width: '100%',
		height: '100%',
		top: 75,
		right: 0,
		left: 75,
		bottom: 100
	},
	colors: ['#a11'],
	curveType: 'function',
	height: 600,
	hAxis: {
		textStyle: {
			fontSize: 12
		},
		slantedText: false,
		maxAlternation: 1
	},
	interpolateNulls: false,
	legend: {
		position: 'none',
	},
	vAxis: {
		minValue: 0,
		textStyle: {
			fontSize: 12
		},
		gridlines: {
			count: 11
		},
		minorGridlines: {
			count: 1
		},
		titleTextStyle: {
			fontSize: 14,
			italic: false
		}
	},
	pointShape: 'diamond',
	pointsVisible: true
}

function HistoryCtrl($scope, $http, $routeParams) {
	routeParams = $routeParams.params.split('|');

	var params = {
		component: routeParams[0],
		testCase: routeParams[1],
		metric: routeParams[2]
	};

	$scope.title = params.component + ' : ' + params.testCase;

	$http.get('api/v1/timeline', {params: params}).then(function(response) {
		var chartData = new google.visualization.DataTable();
		chartData.addColumn('string');
		chartData.addColumn('number');
		chartData.addRows(response.data);

		var div = document.createElement('div');
		div.id = 'history';
		$('#charts').append(div);

		chartOptions.title = params.metric;

		var lineChart = new google.visualization.LineChart(document.getElementById(div.id));
		lineChart.draw(chartData, chartOptions);
	});

	$http.get('api/v1/history', {params: params}).then(function(response) {
		$scope.history = response.data;
	});
}
