var chartOptions = {
	hAxis: {
		textStyle: {
			fontSize: 12
		},
		slantedText: false,
		maxAlternation: 1
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
	legend: {
		position: 'none',
	},
	chartArea: {
		width: '85%'
	},
	colors: ['#a11'],
	curveType: 'function',
	pointsVisible: true,
	interpolateNulls: false,
	pointShape: 'diamond',
	height: 700
}

function drawHistory(component, testCase, metric) {
	var data = JSON.stringify({component: component, testCase: testCase, metric: metric});

	$.post('api/v1/history', data, function(response) {
		var chartData = new google.visualization.DataTable();
		chartData.addColumn('string');
		chartData.addColumn('number');
		chartData.addRows(response);

		var div = document.createElement('div');
		div.id = 'history';
		$('#charts').append(div);

		chartOptions.title = testCase;
		chartOptions.vAxis.title = metric;

		var lineChart = new google.visualization.LineChart(document.getElementById(div.id));
		lineChart.draw(chartData, chartOptions);
	});
}
