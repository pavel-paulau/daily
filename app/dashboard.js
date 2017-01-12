angular
	.module('daily', ['ngRoute'])
	.service('historyParams', function() {
		var component, testCase, metric;

		return {
			getParams: function() {
				return {component: component, testCase: testCase, metric: metric};
			},
			setParams: function(_component, _testCase, _metric) {
				component = _component;
				testCase = _testCase;
				metric = _metric;
			}
		}
	})
	.config(function($routeProvider, $locationProvider) {
		$locationProvider.hashPrefix('');
		$routeProvider
			.when('/dashboard', {templateUrl: 'static/dashboard.html', controller: DashboardCtrl})
			.when('/history', {templateUrl: 'static/history.html', controller: HistoryCtrl})
			.otherwise({redirectTo: 'dashboard'});
	});

function DashboardCtrl($scope, $http, historyParams) {
	$( "#dashboard" ).show();

	$http.get('api/v1/builds').then(function(response) {
		$scope.builds = response.data;
		if ($scope.builds.indexOf('4.6.0-3573') !== -1) {
			$scope.lhb = '4.6.0-3573';
		} else {
			$scope.lhb = $scope.builds[0];
		};
		$scope.rhb = $scope.builds[$scope.builds.length - 1];
	});

	$scope.$watch('lhb', function() {
		Compare($scope, $http);
	});

	$scope.$watch('rhb', function() {
		Compare($scope, $http);
	});

	$scope.evalStatus = evalStatus;

	$scope.getValue = getValue;

	$scope.calcDelta = calcDelta;

	$scope.getReports = getReports;

	$scope.showHistory = function(component, testCase, metric) {
		historyParams.setParams(component, testCase, metric);
	};
}

function Compare($scope, $http) {
	if ( $scope.lhb === undefined || $scope.rhb === undefined ) {
		return;
	}

	$http.get('api/v1/comparison/' + $scope.lhb + '/' + $scope.rhb).then(function(response) {
		$scope.comparisons = response.data;
	});
}

function evalStatus(results, threshold) {
	if (results.length === 1) {
		return "Incomplete";
	}

	var delta = 100 * (results[1].value / results[0].value - 1);

	if (threshold < 0 && delta < threshold) {
		return "Failed"
	}
	if (threshold > 0 && delta > threshold) {
		return "Failed"
	}

	return "Passed";
}

function getValue(results, build) {
	for (var i = 0; i < results.length; i++) {
		if (results[i].build === build) {
			return results[i].value.toLocaleString();
		}
	}
	return '';
}

function calcDelta(results) {
	var delta = 100 * (results[1].value / results[0].value - 1);
	if (delta > 0) {
		return "+" + delta.toFixed(1);
	}
	return delta.toFixed(1);
}

function getReports(results) {
	var reports = [];

	if (results[0].snapshots.length !== results[1].snapshots.length) {
		return reports;
	}

	for (var i = 0; i < results.length; i++) {
		for (var j = 0; j < results[i].snapshots.length; j++) {
			var snapshot = results[i].snapshots[j];
			var report = 'http://cbmonitor.sc.couchbase.com/reports/html/?snapshot=' + snapshot;
			reports.push(report);
		}
	}

	return reports;
}
