package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
)

type WebNode struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Type     string `json:"type"`
	Group    string `json:"group"`
	Status   string `json:"status"`
	Tooltip  string `json:"title"`
}

type WebEdge struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Label string `json:"label"`
}

type WebGraph struct {
	Nodes []WebNode `json:"nodes"`
	Edges []WebEdge `json:"edges"`
}

func generateWebGraph(channels map[string]*ChannelInfo) WebGraph {
	var graph WebGraph

	graph.Nodes = append(graph.Nodes, WebNode{
		ID:      "main",
		Label:   "Main",
		Type:    "program",
		Group:   "main",
		Status:  "normal",
		Tooltip: "Main program",
	})

	for name, channel := range channels {
		status := "normal"
		tooltip := fmt.Sprintf("Type: %s\nDeclaration: %s", channel.Type, channel.Declaration)
		
		sendCount := 0
		receiveCount := 0
		
		for _, op := range channel.SendOps {
			if op != "" {
				sendCount++
			}
		}
		
		for _, op := range channel.ReceiveOps {
			if op != "" {
				receiveCount++
			}
		}
		
		if sendCount == 0 && receiveCount == 0 {
			status = "dangling"
			tooltip += "\n⚠️ Dangling channel: No send or receive operations"
		} else if sendCount == 0 {
			status = "receive-only"
			tooltip += "\n⚠️ Receive-only channel: No send operations"
		} else if receiveCount == 0 {
			status = "send-only"
			tooltip += "\n⚠️ Send-only channel: No receive operations"
		}

		graph.Nodes = append(graph.Nodes, WebNode{
			ID:      name,
			Label:   name,
			Type:    channel.Type,
			Group:   "channel",
			Status:  status,
			Tooltip: tooltip,
		})

		for range channel.SendOps {
			graph.Edges = append(graph.Edges, WebEdge{
				From:  "main",
				To:    name,
				Label: "send",
			})
		}

		for range channel.ReceiveOps {
			graph.Edges = append(graph.Edges, WebEdge{
				From:  name,
				To:    "main",
				Label: "receive",
			})
		}
	}

	return graph
}

func startWebServer(channels map[string]*ChannelInfo) {
	graph := generateWebGraph(channels)

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl := `
<!DOCTYPE html>
<html>
<head>
    <title>Channel Flow Visualization</title>
    <script type="text/javascript" src="https://unpkg.com/vis-network/standalone/umd/vis-network.min.js"></script>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600&display=swap" rel="stylesheet">
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
            font-family: 'Inter', sans-serif;
        }

        body {
            background-color: #f8f9fa;
            color: #2c3e50;
            line-height: 1.6;
        }

        .container {
            max-width: 1400px;
            margin: 0 auto;
            padding: 20px;
        }

        .header {
            background: white;
            padding: 20px;
            border-radius: 12px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.05);
            margin-bottom: 20px;
        }

        .header h1 {
            font-size: 24px;
            font-weight: 600;
            color: #1a1a1a;
            margin-bottom: 8px;
        }

        .header p {
            color: #666;
            font-size: 14px;
        }

        .controls {
            display: grid;
            grid-template-columns: 1fr 300px;
            gap: 20px;
            margin-bottom: 20px;
        }

        .main-controls {
            background: white;
            padding: 20px;
            border-radius: 12px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.05);
        }

        .button-group {
            display: flex;
            gap: 10px;
            margin-bottom: 20px;
        }

        .button {
            padding: 8px 16px;
            background: #4a90e2;
            color: white;
            border: none;
            border-radius: 6px;
            cursor: pointer;
            font-size: 14px;
            font-weight: 500;
            transition: all 0.2s ease;
        }

        .button:hover {
            background: #357abd;
            transform: translateY(-1px);
        }

        .button:active {
            transform: translateY(0);
        }

        .sidebar {
            background: white;
            padding: 20px;
            border-radius: 12px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.05);
        }

        .filter-controls {
            margin-bottom: 20px;
        }

        .filter-controls h3 {
            font-size: 16px;
            font-weight: 600;
            margin-bottom: 12px;
            color: #1a1a1a;
        }

        .filter-group {
            display: flex;
            flex-direction: column;
            gap: 8px;
        }

        .filter-item {
            display: flex;
            align-items: center;
            gap: 8px;
        }

        .filter-checkbox {
            width: 16px;
            height: 16px;
            accent-color: #4a90e2;
        }

        .filter-label {
            font-size: 14px;
            color: #4a4a4a;
        }

        .legend {
            margin-top: 20px;
        }

        .legend h3 {
            font-size: 16px;
            font-weight: 600;
            margin-bottom: 12px;
            color: #1a1a1a;
        }

        .legend-item {
            display: flex;
            align-items: center;
            gap: 10px;
            margin-bottom: 8px;
        }

        .legend-color {
            width: 16px;
            height: 16px;
            border-radius: 4px;
            border: 1px solid rgba(0,0,0,0.1);
        }

        .legend-label {
            font-size: 14px;
            color: #4a4a4a;
        }

        #network {
            background: white;
            border-radius: 12px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.05);
            height: 800px;
        }

        @media (max-width: 1200px) {
            .controls {
                grid-template-columns: 1fr;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Channel Flow Visualization</h1>
            <p>Interactive visualization of Go channel usage patterns</p>
        </div>

        <div class="controls">
            <div class="main-controls">
                <div class="button-group">
                    <button class="button" onclick="stabilize()">Stabilize Layout</button>
                    <button class="button" onclick="togglePhysics()">Toggle Physics</button>
                    <button class="button" onclick="resetView()">Reset View</button>
                </div>
                <div id="network"></div>
            </div>

            <div class="sidebar">
                <div class="filter-controls">
                    <h3>Filter Channels</h3>
                    <div class="filter-group">
                        <div class="filter-item">
                            <input type="checkbox" id="showNormal" class="filter-checkbox" checked onchange="updateFilters()">
                            <label for="showNormal" class="filter-label">Normal Channels</label>
                        </div>
                        <div class="filter-item">
                            <input type="checkbox" id="showDangling" class="filter-checkbox" checked onchange="updateFilters()">
                            <label for="showDangling" class="filter-label">Dangling Channels</label>
                        </div>
                        <div class="filter-item">
                            <input type="checkbox" id="showReceiveOnly" class="filter-checkbox" checked onchange="updateFilters()">
                            <label for="showReceiveOnly" class="filter-label">Receive-only Channels</label>
                        </div>
                        <div class="filter-item">
                            <input type="checkbox" id="showSendOnly" class="filter-checkbox" checked onchange="updateFilters()">
                            <label for="showSendOnly" class="filter-label">Send-only Channels</label>
                        </div>
                    </div>
                </div>

                <div class="legend">
                    <h3>Channel Status</h3>
                    <div class="legend-item">
                        <div class="legend-color" style="background: #D2E5FF;"></div>
                        <span class="legend-label">Normal Channel</span>
                    </div>
                    <div class="legend-item">
                        <div class="legend-color" style="background: #FFB1B1;"></div>
                        <span class="legend-label">Dangling Channel</span>
                    </div>
                    <div class="legend-item">
                        <div class="legend-color" style="background: #FFD700;"></div>
                        <span class="legend-label">Receive-only Channel</span>
                    </div>
                    <div class="legend-item">
                        <div class="legend-color" style="background: #98FB98;"></div>
                        <span class="legend-label">Send-only Channel</span>
                    </div>
                </div>
            </div>
        </div>
    </div>

    <script>
        const nodes = new vis.DataSet({{.Nodes}});
        const edges = new vis.DataSet({{.Edges}});
        
        const container = document.getElementById('network');
        const data = { nodes, edges };
        const options = {
            nodes: {
                shape: 'box',
                margin: 10,
                font: { size: 14 },
                color: {
                    background: function(node) {
                        switch(node.status) {
                            case 'dangling': return '#FFB1B1';
                            case 'receive-only': return '#FFD700';
                            case 'send-only': return '#98FB98';
                            default: return '#D2E5FF';
                        }
                    },
                    border: '#2B7CE9',
                    highlight: { background: '#FFB1B1', border: '#FF0000' }
                },
                widthConstraint: {
                    minimum: 100,
                    maximum: 200
                },
                heightConstraint: {
                    minimum: 30,
                    maximum: 50
                }
            },
            edges: {
                arrows: { to: { enabled: true, scaleFactor: 1 } },
                font: { size: 12, align: 'middle' },
                color: { color: '#848484', highlight: '#FF0000' },
                smooth: {
                    type: 'continuous',
                    forceDirection: 'none',
                    roundness: 0.5
                }
            },
            physics: {
                enabled: true,
                stabilization: {
                    enabled: true,
                    iterations: 1000,
                    updateInterval: 50,
                    fit: true
                },
                barnesHut: {
                    gravitationalConstant: -1000,
                    centralGravity: 0.05,
                    springLength: 200,
                    springConstant: 0.01,
                    damping: 0.15,
                    avoidOverlap: 0.5
                },
                maxVelocity: 30,
                minVelocity: 0.2,
                solver: 'barnesHut',
                timestep: 0.3
            },
            layout: {
                improvedLayout: true,
                hierarchical: {
                    enabled: false,
                    direction: 'UD',
                    sortMethod: 'directed'
                }
            },
            interaction: {
                dragNodes: true,
                dragView: true,
                zoomView: true,
                hover: true,
                tooltipDelay: 200,
                hideEdgesOnDrag: true,
                hideEdgesOnZoom: true
            }
        };
        
        const network = new vis.Network(container, data, options);

        // Add event listeners
        network.on("stabilizationProgress", function(params) {
            console.log('Stabilization progress:', params.iterations, '/', params.total);
        });

        network.on("stabilizationIterationsDone", function() {
            console.log('Stabilization finished');
        });

        // Control functions
        function stabilize() {
            network.stabilize(100);
        }

        function togglePhysics() {
            options.physics.enabled = !options.physics.enabled;
            network.setOptions(options);
        }

        function resetView() {
            network.fit({
                animation: {
                    duration: 1000,
                    easingFunction: 'easeInOutQuad'
                }
            });
        }

        function updateFilters() {
            const showNormal = document.getElementById('showNormal').checked;
            const showDangling = document.getElementById('showDangling').checked;
            const showReceiveOnly = document.getElementById('showReceiveOnly').checked;
            const showSendOnly = document.getElementById('showSendOnly').checked;

            nodes.forEach(node => {
                if (node.id === 'main') {
                    node.hidden = false;
                    return;
                }

                switch(node.status) {
                    case 'normal':
                        node.hidden = !showNormal;
                        break;
                    case 'dangling':
                        node.hidden = !showDangling;
                        break;
                    case 'receive-only':
                        node.hidden = !showReceiveOnly;
                        break;
                    case 'send-only':
                        node.hidden = !showSendOnly;
                        break;
                }
            });

            network.setData({ nodes, edges });
        }

        // Initial stabilization
        network.stabilize(100);
    </script>
</body>
</html>`

		t, err := template.New("visualization").Parse(tmpl)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		nodesJSON, _ := json.Marshal(graph.Nodes)
		edgesJSON, _ := json.Marshal(graph.Edges)

		data := struct {
			Nodes template.JS
			Edges template.JS
		}{
			Nodes: template.JS(nodesJSON),
			Edges: template.JS(edgesJSON),
		}

		if err := t.Execute(w, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	fmt.Println("\nStarting web server on http://localhost:8080")
	fmt.Println("Open your browser to view the interactive visualization")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("Error starting server: %v\n", err)
	}
} 